<script setup lang="ts">
import Dialog from 'primevue/dialog';
import Button from 'primevue/button';
import { onMounted, ref, watch } from "vue";
import { ScanPathItem } from "../entries/entries";
import { FileFilter } from "../../bindings/github.com/wailsapp/wails/v3/pkg/application";
import { AppService } from "../../bindings/github.com/wux1an/wxapkg";

const emit = defineEmits<{
  confirm: [path: ScanPathItem];
}>();

const visible = defineModel('visible', {
  default: false,
});
const loading = ref(false);
const selectedPath = ref<ScanPathItem | null>(null);
const defaultPaths = ref<ScanPathItem[]>([]);
const activeTab = ref<'auto' | 'manual'>('auto');

function selectWxapkgFile() {
  AppService.OpenFileDialog("选择微信小程序文件(.wxapkg)", "", [
    {
      DisplayName: "微信小程序文件",
      Pattern: "*.wxapkg",
    },
    {
      DisplayName: "所有文件",
      Pattern: "*.*",
    }
  ] as FileFilter[]).then(path => {
    selectedPath.value = new ScanPathItem(path, false);
    // 手动指定模式直接添加
    if (selectedPath.value) {
      emit('confirm', selectedPath.value);
      visible.value = false;
    }
  });
}

function selectAppDir() {
  AppService.OpenDirectoryDialog("选择小程序目录", "").then(path => {
    selectedPath.value = new ScanPathItem(path, false);
    // 手动指定模式直接添加
    if (selectedPath.value) {
      emit('confirm', selectedPath.value);
      visible.value = false;
    }
  });
}

function selectInstallDir() {
  AppService.OpenDirectoryDialog("选择微信小程序安装目录", "").then(path => {
    selectedPath.value = new ScanPathItem(path, true);
    // 手动指定模式直接添加
    if (selectedPath.value) {
      emit('confirm', selectedPath.value);
      visible.value = false;
    }
  });
}

function selectDefaultPath(path: ScanPathItem) {
  selectedPath.value = path;
  // 直接开始扫描
  if (selectedPath.value) {
    emit('confirm', selectedPath.value);
    visible.value = false;
  }
}

function confirmSelection() {
  if (selectedPath.value) {
    emit('confirm', selectedPath.value);
    visible.value = false;
  }
}

onMounted(() => {
  loadDefaultPaths();
})

// 监听对话框打开状态
watch(visible, (newValue) => {
  if (newValue) {
    // 对话框打开时重新加载默认路径
    loadDefaultPaths();
  }
})

function loadDefaultPaths() {
  loading.value = true;
  activeTab.value = 'auto'; // 重置为自动扫描标签
  AppService.GetDefaultPaths()
    .then(value => {
      defaultPaths.value = value.map(item => new ScanPathItem(item, true));
    })
    .finally(() => {
      loading.value = false;
      // 加载完成后立刻检查，没有默认目录就切换到手动指定
      if (defaultPaths.value.length === 0) {
        activeTab.value = 'manual';
      }
    });
}
</script>

<template>
  <Dialog
    v-model:visible="visible"
    modal
    header="扫描微信小程序"
    :style="{ width: '50rem' }"
  >
    <!-- Tab Navigation -->
    <div class="flex border-b border-gray-200 mb-6">
      <button
        :class="[
          'px-6 py-3 text-sm font-medium transition-colors relative',
          activeTab === 'auto'
            ? 'text-blue-600'
            : 'text-gray-500 hover:text-gray-700'
        ]"
        @click="activeTab = 'auto'"
      >
        自动扫描
        <div v-if="activeTab === 'auto'" class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600"></div>
      </button>
      <button
        :class="[
          'px-6 py-3 text-sm font-medium transition-colors relative',
          activeTab === 'manual'
            ? 'text-blue-600'
            : 'text-gray-500 hover:text-gray-700'
        ]"
        @click="activeTab = 'manual'"
      >
        手动指定
        <div v-if="activeTab === 'manual'" class="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600"></div>
      </button>
    </div>

    <!-- 自动扫描标签页 -->
    <div v-if="activeTab === 'auto'" class="h-80 flex flex-col">
      <div v-if="loading" class="text-center py-12 text-gray-500">
        <i class="pi pi-spin pi-spinner text-3xl mb-3"></i>
        <p>正在检测微信小程序安装目录...</p>
      </div>

      <div v-else-if="defaultPaths.length === 0" class="text-center py-12 text-gray-500">
        <i class="pi pi-exclamation-triangle text-3xl mb-3 text-yellow-500"></i>
        <p class="font-medium">未找到微信小程序安装目录</p>
        <p class="text-sm mt-2">请使用手动指定模式</p>
      </div>

      <div v-else class="flex-1 flex flex-col">
        <div class="mb-3">
          <h3 class="text-sm font-medium text-gray-700">检测到的小程序安装目录</h3>
        </div>

        <div class="space-y-2 overflow-y-auto flex-1">
          <div
            v-for="path in defaultPaths"
            :key="path.path"
            class="p-3 rounded-lg border cursor-pointer transition-all border-gray-200 hover:border-yellow-300 hover:bg-yellow-50"
            @click="selectDefaultPath(path)"
          >
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded-lg bg-yellow-100 flex items-center justify-center flex-shrink-0">
                <i class="pi pi-folder text-yellow-600"></i>
              </div>
              <div class="flex-1 min-w-0">
                <div class="font-mono text-sm truncate">{{ path.path }}</div>
                <div class="text-xs text-gray-500 mt-1">点击开始扫描</div>
              </div>
              <i class="pi pi-chevron-right text-gray-400"></i>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 手动指定标签页 -->
    <div v-else class="h-80 flex flex-col">
      <div class="grid grid-cols-3 gap-3 mb-4">
        <!-- 模式1: 安装目录（放在首位） -->
        <div
          class="p-4 rounded-lg border-2 cursor-pointer transition-all text-center border-gray-200 hover:border-yellow-300 hover:bg-yellow-50"
          @click="selectInstallDir"
        >
          <div class="w-12 h-12 mx-auto mb-3 rounded-lg bg-yellow-100 flex items-center justify-center">
            <i class="pi pi-users text-yellow-600 text-xl"></i>
          </div>
          <div class="font-medium text-gray-800 text-sm mb-1">安装目录</div>
          <div class="text-xs text-gray-500 mb-2">自动识别所有小程序</div>
          <div class="text-xs text-green-500 bg-green-50 rounded py-1 px-2">
            自动获取密钥
          </div>
        </div>

        <!-- 模式2: wxapkg文件 -->
        <div
          class="p-4 rounded-lg border-2 cursor-pointer transition-all text-center border-gray-200 hover:border-blue-300 hover:bg-blue-50"
          @click="selectWxapkgFile"
        >
          <div class="w-12 h-12 mx-auto mb-3 rounded-lg bg-blue-100 flex items-center justify-center">
            <i class="pi pi-file text-blue-600 text-xl"></i>
          </div>
          <div class="font-medium text-gray-800 text-sm mb-1">.wxapkg 文件</div>
          <div class="text-xs text-gray-500 mb-2">选择单个文件</div>
          <div class="text-xs text-orange-500 bg-orange-50 rounded py-1 px-2">
            需手动输入密钥
          </div>
        </div>

        <!-- 模式3: 小程序目录 -->
        <div
          class="p-4 rounded-lg border-2 cursor-pointer transition-all text-center border-gray-200 hover:border-green-300 hover:bg-green-50"
          @click="selectAppDir"
        >
          <div class="w-12 h-12 mx-auto mb-3 rounded-lg bg-green-100 flex items-center justify-center">
            <i class="pi pi-folder-open text-green-600 text-xl"></i>
          </div>
          <div class="font-medium text-gray-800 text-sm mb-1">小程序目录</div>
          <div class="text-xs text-gray-500 mb-2">选择小程序文件夹</div>
          <div class="text-xs text-orange-500 bg-orange-50 rounded py-1 px-2">
            需手动输入密钥
          </div>
        </div>
      </div>

      <!-- 说明信息 -->
      <div class="bg-gray-50 rounded-lg p-3 text-sm text-gray-600 mb-auto">
        <div class="flex items-start gap-2">
          <div class="flex-1">
            <div class="font-medium text-sm mb-2 text-gray-700">关于解密密钥</div>
            <div class="text-xs space-y-1">
              <div><strong>密钥即小程序ID</strong>，格式：<code class="bg-gray-200 px-1 rounded mx-1">wx</code> 开头 + 16位字符</div>
              <div><strong>示例：</strong><code class="bg-gray-200 px-1 rounded mx-1">wxabcdef1234567890</code></div>
              <div class="text-gray-500 mt-2">安装目录模式会自动识别小程序ID作为密钥，其他模式需要在解包时手动输入</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 底部操作栏 -->
    <div class="flex justify-end pt-6 mt-6 border-t border-gray-200">
      <Button
        label="取消"
        severity="secondary"
        outlined
        @click="visible = false"
      />
    </div>
  </Dialog>
</template>

<style scoped>
</style>