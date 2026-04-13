<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import ScanDialog from "./components/ScanDialog.vue";
import { EventUnpackProgress, ScanPathItem } from "./entries/entries";
import Button from "primevue/button";
import InputText from "primevue/inputtext";
import DataTable from 'primevue/datatable';
import Column from 'primevue/column';
import IconField from 'primevue/iconfield';
import InputIcon from 'primevue/inputicon';
import { formatSize, formatTime, UnpackStatusType, useAppToast, getUnpackButtonProps } from "./entries/util";
import Toast from 'primevue/toast';
import { AppService } from "../bindings/github.com/wux1an/wxapkg/index"
import { UnpackOptions, WxapkgItem } from "../bindings/github.com/wux1an/wxapkg/wechat"
import { Events } from '@wailsio/runtime'
import UnpackDialog from "./components/UnpackDialog.vue";

const scanDialogVisible = ref(false)
const unpackDialogVisible = ref(false)
const search = ref<string>('')
const wxapkgItems = ref<WxapkgItem[]>([]);
const toast = useAppToast()
const version = ref<string>('v0.0.0')
const github = ref<string>('https://github.com')
const selectedWxapkgItem = ref<WxapkgItem | null>(null);

// 添加强制刷新的key
const tableKey = ref<string>('main-table')

const filteredItems = computed(() => {
  if (!search || !search.value.trim()) {
    return wxapkgItems.value
  }

  const queryStr = search.value.toLowerCase().trim()
  return wxapkgItems.value.filter(item =>
    item.WxId.toLowerCase().includes(queryStr) ||
    item.Location.toLowerCase().includes(queryStr)
  )
})

function openUrl(url: string) {
  AppService.OpenUrl(url)
}

function openFolder(folder: string) {
  AppService.OpenPath(folder).catch(e => toast.error('打开目录失败', e))
}

function confirmScan(path: ScanPathItem) {
  AppService.ScanWxapkgItem(path.path, path.scan)
    .then((v: WxapkgItem[]) => {
      wxapkgItems.value = v
      toast.info('扫描小程序完成', `共 ${v.length} 个结果`)
    })
    .catch(e => toast.error('扫描小程序出错', e))
}

function copyPath(path: string) {
  AppService.ClipboardSetText(path)
    .then(() => {
      toast.info('成功', '复制路径成功')
    }).catch(() => toast.error('失败', '复制路径失败'))
}

function unpack(item: WxapkgItem) {
  selectedWxapkgItem.value = item
  unpackDialogVisible.value = true
}

function confirmUnpack(options: UnpackOptions) {
  if (selectedWxapkgItem.value) {
    AppService.UnpackWxapkgItem(selectedWxapkgItem.value, options)
  }
}

function openUnpackResultDirectory(path: string) {
  openFolder(path)
}

function handleDialogHide() {
  console.log('对话框关闭，最终状态同步:', selectedWxapkgItem.value?.UnpackStatus)

  // 对话框关闭时，确保状态同步
  if (selectedWxapkgItem.value) {
    const index = wxapkgItems.value.findIndex(item => item.UUID === selectedWxapkgItem.value.UUID)
    if (index !== -1) {
      console.log('强制同步状态到表格:', selectedWxapkgItem.value.UnpackStatus)

      // 使用map创建新数组，确保响应式
      wxapkgItems.value = wxapkgItems.value.map((item, i) =>
        i === index ? { ...item, ...selectedWxapkgItem.value } : item
      )
    }
  }
  selectedWxapkgItem.value = null
}

function clearAll() {
  wxapkgItems.value = []
  toast.info('清空', '已清空所有小程序')
}

// 每个UUID的防抖定时器
const pendingTimers = new Map<string, ReturnType<typeof setTimeout>>()
// 已发送过toast的UUID（防止重复弹toast）
const notifiedUuids = new Set<string>()

function processProgress(uuid: string) {
  AppService.GetWxapkgItem(uuid).then(data => {
    if (!data) return

    const index = wxapkgItems.value.findIndex(item => item.UUID === uuid)

    if (index !== -1) {
      const currentItem = wxapkgItems.value[index]
      const currentStatus = currentItem.UnpackStatus

      // 状态保护：不允许从 finished/error 回退到 running
      if ((currentStatus === UnpackStatusType.Finished || currentStatus === UnpackStatusType.Error)
          && data.UnpackStatus === UnpackStatusType.Running) {
        return
      }

      // 完全替换对象
      const updatedItem: WxapkgItem = { ...currentItem, ...data }

      // 替换数组元素，强制响应式更新
      wxapkgItems.value = wxapkgItems.value.map((item, i) =>
        i === index ? updatedItem : item
      )

      // 强制刷新DataTable
      tableKey.value = `table-${Date.now()}-${data.UnpackStatus}`

      // 如果对话框正在显示这个项目，同步更新
      if (unpackDialogVisible.value && selectedWxapkgItem.value?.UUID === uuid) {
        selectedWxapkgItem.value = updatedItem
      }

      // toast只弹一次（对话框不显示时才弹）
      if (!notifiedUuids.has(uuid)) {
        if (data.UnpackStatus === UnpackStatusType.Finished && !unpackDialogVisible.value) {
          notifiedUuids.add(uuid)
          toast.info('解包完成', `输出路径：${data.UnpackSavePath}`)
        } else if (data.UnpackStatus === UnpackStatusType.Error) {
          notifiedUuids.add(uuid)
          toast.error('解包失败', `${data.UnpackErrorMessage}`)
        }
      }
    }
  })
}

onMounted(() => {
  Events.On(EventUnpackProgress, callback => {
    const uuid = callback.data as string

    // 清除该UUID已有的定时器，重新计时100ms
    const existing = pendingTimers.get(uuid)
    if (existing) clearTimeout(existing)

    const timer = setTimeout(() => {
      pendingTimers.delete(uuid)
      processProgress(uuid)
    }, 100)

    pendingTimers.set(uuid, timer)
  })

  AppService.Version().then(v => version.value = v)
  AppService.Github().then(v => github.value = v)
})

onBeforeUnmount(() => {
  Events.Off(EventUnpackProgress)
  // 清理所有待处理的定时器
  for (const t of pendingTimers.values()) clearTimeout(t)
  pendingTimers.clear()
})
</script>

<template>
  <div class="flex flex-col h-screen bg-white">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 border-b border-gray-200">
      <div class="flex items-center space-x-4">
        <h1 class="text-xl font-bold text-gray-800">微信小程序解包工具</h1>
        <span class="text-sm text-gray-500">{{ version }}</span>
      </div>
      <div class="flex items-center space-x-2 text-sm text-gray-500">
        <span class="hover:underline cursor-pointer" @click="openUrl(github)" v-tooltip="github">GitHub</span>
      </div>
    </div>

    <!-- Search and Actions Bar -->
    <div class="flex items-center justify-between px-6 py-4">
      <IconField class="flex-1 max-w-2xl">
        <InputIcon class="pi pi-search" />
        <InputText v-model="search" placeholder="搜索小程序 ID 或路径" fluid />
      </IconField>
      <div class="flex items-center gap-3">
        <Button
          label="扫描小程序"
          icon="pi pi-folder-open"
          severity="primary"
          @click="scanDialogVisible = true"
        />
        <Button
          v-if="wxapkgItems.length > 0"
          label="清空列表"
          icon="pi pi-trash"
          severity="secondary"
          outlined
          @click="clearAll"
        />
      </div>
    </div>

    <!-- Data Table -->
    <div class="flex-1 px-6 pb-6 overflow-hidden">
      <DataTable
        :value="filteredItems"
        data-key="UUID"
        sortField="LastModifyTime"
        :sortOrder="-1"
        scrollable
        scrollHeight="flex"
        tableStyle="min-width: 50rem; table-layout: fixed"
        class="font-mono h-full"
        size="normal"
        :key="tableKey"
      >
        <template #empty>
          <div class="flex justify-center items-center h-full text-gray-400">
            {{ search ? `没有搜索到与 '${search}' 相关的小程序` : '没有小程序，请扫描或添加' }}
          </div>
        </template>

        <Column header="小程序ID" field="WxId" style="width: 170px" class="user-select"/>
        <Column header="修改时间" field="LastModifyTime" :sortable="true" style="width: 180px">
          <template #body="{ data }">
            <div class="text-nowrap">{{ formatTime(data.LastModifyTime, false) }}</div>
          </template>
        </Column>
        <Column header="大小" field="Size" style="width: 100px" headerClass="text-right">
          <template #body="{ data }">
            <div class="text-right text-nowrap">{{ formatSize(data.Size) }}</div>
          </template>
        </Column>
        <Column header="路径" field="Location">
          <template #body="{ data }">
            <div
              class="ellipsis-left overflow-hidden text-ellipsis whitespace-nowrap cursor-default"
              v-tooltip.bottom="data.Location + '\n点击复制'"
              @click="copyPath(data.Location)"
            >
              {{ data.Location }}
            </div>
          </template>
        </Column>
        <Column header="解包" style="width: 70px" headerClass="text-center" bodyClass="text-center">
          <template #body="{ data }">
              <Button
                v-if="data.UnpackStatus === UnpackStatusType.Running"
                v-tooltip.top="`解包中 ${Math.round(data.UnpackProgress)}%`"
                icon="pi pi-spin pi-spinner"
                disabled
                class="!text-blue-500 !bg-transparent !border-0 !p-0 !min-w-0 w-7 h-7 justify-center"
                severity="text"
                rounded
                size="small"
              />
              <!-- 已完成状态 -->
              <Button
                v-else-if="data.UnpackStatus === UnpackStatusType.Finished"
                v-tooltip.top="'打开目录'"
                icon="pi pi-folder-open"
                @click="openFolder(data.UnpackSavePath)"
                class="!text-green-500 !bg-transparent !border-0 !p-0 !min-w-0 w-7 h-7 justify-center"
                severity="text"
                rounded
                size="small"
              />
              <!-- 错误状态 -->
              <Button
                v-else-if="data.UnpackStatus === UnpackStatusType.Error"
                v-tooltip.top="data.UnpackErrorMessage || '解包失败'"
                icon="pi pi-times"
                disabled
                class="!text-red-500 !bg-transparent !border-0 !p-0 !min-w-0 w-7 h-7 justify-center"
                severity="text"
                rounded
                size="small"
              />
              <!-- 默认状态 -->
              <Button
                v-else
                v-tooltip.top="'解包'"
                icon="pi pi-box"
                @click="unpack(data)"
                class="!text-gray-500 !bg-transparent !border-0 !p-0 !min-w-0 w-7 h-7 justify-center"
                severity="text"
                rounded
                size="small"
              />
          </template>
        </Column>
      </DataTable>
    </div>

    <!-- Footer -->
    <div class="text-center text-xs text-gray-500 py-3 border-t border-gray-200">
      本工具仅供学习研究使用，请遵守相关法律法规，不得用于非法用途
    </div>

    <!-- Dialogs -->
    <ScanDialog v-model:visible="scanDialogVisible" @confirm="confirmScan"/>
    <UnpackDialog
      v-model:visible="unpackDialogVisible"
      v-model:item="selectedWxapkgItem"
      @after-hide="handleDialogHide"
      @confirm="confirmUnpack"
      @open-directory="openUnpackResultDirectory"
    />
    <Toast position="bottom-right" />
  </div>
</template>

<style scoped>
::v-deep(.p-datatable-table-container) {
  scrollbar-width: none;
}

::v-deep(.p-datatable-column-sorted) {
  background: transparent;
}

.ellipsis-left {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  direction: rtl;
  text-align: left;
}
</style>