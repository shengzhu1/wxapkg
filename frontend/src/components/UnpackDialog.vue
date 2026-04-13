<script setup lang="ts">
import Dialog from 'primevue/dialog';
import { reactive, ref, watch, computed } from "vue";
import { UnpackOptions, WxapkgItem } from "../../bindings/github.com/wux1an/wxapkg/wechat";
import InputText from "primevue/inputtext";
import Checkbox from "primevue/checkbox";
import Button from "primevue/button";
import IconField from 'primevue/iconfield';
import InputIcon from 'primevue/inputicon';
import ProgressBar from 'primevue/progressbar';
import { AppService } from "../../bindings/github.com/wux1an/wxapkg";
import { UnpackStatusType, formatProgress } from "../entries/util";

type DialogStage = 'config' | 'progress' | 'complete' | 'error';

const emit = defineEmits<{
  confirm: [options: UnpackOptions];
  afterHide: [];
  openDirectory: [path: string];
}>();

const visible = defineModel<boolean>('visible', { default: false });
const item = defineModel<WxapkgItem | null>('item', { default: null });

const options = reactive<UnpackOptions>({
  EnableDecrypt: true,
  EnableHtmlBeautify: false,
  EnableJsBeautify: false,
  EnableJsonBeautify: true,
  OutputDir: "",
});

// 添加解密密钥状态
const decryptKey = ref('');

const currentStage = ref<DialogStage>('config');
const currentProgress = ref<WxapkgItem | null>(null);

// 真实输出目录：后端根据平台计算
const actualOutputDir = ref('')

// 监听OutputDir变化，重新计算实际输出路径
watch(() => [options.OutputDir, item.value?.Location], async () => {
  if (options.OutputDir && item.value) {
    actualOutputDir.value = await AppService.ComputeSavePath(options.OutputDir, item.value.Location)
  } else {
    actualOutputDir.value = ''
  }
}, { immediate: true })

// Reset dialog when opening
watch(visible, (newVal) => {
  if (newVal && item.value) {
    currentStage.value = 'config';
    currentProgress.value = item.value;
    options.OutputDir = '';

    // 设置解密相关：默认启用解密
    options.EnableDecrypt = true;

    // 设置解密密钥：优先使用已知的WxId，否则使用EncryptKey
    if (item.value.WxId && item.value.WxId.startsWith('wx')) {
      decryptKey.value = item.value.WxId;
    } else if (item.value.EncryptKey) {
      decryptKey.value = item.value.EncryptKey;
    } else {
      decryptKey.value = '';
    }

    // If already unpacking or finished, skip to appropriate stage
    if (item.value.UnpackStatus === UnpackStatusType.Running) {
      currentStage.value = 'progress';
    } else if (item.value.UnpackStatus === UnpackStatusType.Finished) {
      currentStage.value = 'complete';
    } else if (item.value.UnpackStatus === UnpackStatusType.Error) {
      currentStage.value = 'error';
    }
  }
});

// Watch for progress updates - 监听item的所有变化
watch(() => item.value, (newItem) => {
  if (newItem && visible.value) {
    // 确保使用最新的引用
    currentProgress.value = newItem

    // Auto-transition stages based on status
    if (newItem.UnpackStatus === UnpackStatusType.Finished && currentStage.value === 'progress') {
      currentStage.value = 'complete';
    } else if (newItem.UnpackStatus === UnpackStatusType.Error && currentStage.value === 'progress') {
      currentStage.value = 'error';
    }
  }
}, { deep: true, immediate: true })

// 监听visible变化，确保对话框打开时同步状态
watch(visible, (newVisible) => {
  if (newVisible && item.value) {
    currentProgress.value = item.value
  }
})

function selectFolder() {
  AppService.OpenDirectoryDialog("选择输出目录", options.OutputDir).then((result) => {
    options.OutputDir = result;
  });
}

function startUnpack() {
  // 将解密密钥设置到item中
  if (item.value && decryptKey.value) {
    item.value.EncryptKey = decryptKey.value;
  }
  // 传递真实输出目录
  options.SavePath = actualOutputDir.value;
  emit('confirm', options);
  currentStage.value = 'progress';
}

function openOutputDirectory() {
  if (currentProgress.value?.UnpackSavePath) {
    emit('openDirectory', currentProgress.value.UnpackSavePath);
  }
}

function closeDialog() {
  // 在关闭对话框前，确保状态同步
  if (item.value && currentProgress.value) {
    // 将最新的状态同步回item
    Object.assign(item.value, currentProgress.value)
  }
  visible.value = false;
}

function minimizeDialog() {
  visible.value = false;
}
</script>

<template>
  <!-- Stage 1: Configuration -->
  <Dialog
    v-if="currentStage === 'config'"
    v-model:visible="visible"
    modal
    header="解包配置"
    :style="{ width: '32rem' }"
    @after-hide="emit('afterHide')"
  >
    <!-- 解密设置 -->
    <div class="mb-6">
      <h3 class="text-base font-semibold text-gray-700 mb-3">解密</h3>
      <div class="flex items-center gap-4 mb-3">
        <div class="flex items-center gap-2">
          <Checkbox v-model="options.EnableDecrypt" input-id="decrypt" binary/>
          <label for="decrypt" class="cursor-pointer text-sm">启用解密</label>
        </div>
      </div>
      <div v-if="options.EnableDecrypt" class="space-y-2">
        <InputText
          v-model="decryptKey"
          placeholder="小程序ID，如：wxabcdef1234567890"
          fluid
          class="font-mono"
        />
        <div class="text-xs text-gray-500">
          密钥即小程序ID，格式：wx 开头 + 16位字符
        </div>
      </div>
    </div>

    <!-- 代码美化选项 -->
    <div class="mb-6">
      <h3 class="text-base font-semibold text-gray-700 mb-3">代码美化</h3>
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-2">
          <Checkbox v-model="options.EnableJsonBeautify" input-id="json" binary/>
          <label for="json" class="cursor-pointer text-sm">JSON</label>
        </div>
        <div class="flex items-center gap-2">
          <Checkbox v-model="options.EnableHtmlBeautify" input-id="html" binary/>
          <label for="html" class="cursor-pointer text-sm">HTML</label>
        </div>
        <div class="flex items-center gap-2">
          <Checkbox v-model="options.EnableJsBeautify" input-id="js" binary/>
          <label for="js" class="cursor-pointer text-sm">JavaScript</label>
        </div>
      </div>
    </div>

    <!-- 输出目录设置 -->
    <div class="mb-6">
      <h3 class="text-base font-semibold text-gray-700 mb-3">输出目录</h3>
      <IconField>
        <InputText
          v-model="options.OutputDir"
          id="outputDir"
          placeholder="点击右侧图标选择输出目录"
          fluid
        />
        <InputIcon
          position="right"
          class="pi pi-folder cursor-pointer text-primary"
          @click="selectFolder"
        />
      </IconField>
      <div class="text-xs text-gray-500 mt-1" v-if="options.OutputDir">
        将输出到：{{ actualOutputDir }}
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="flex justify-end gap-2">
      <Button
        label="取消"
        severity="secondary"
        @click="closeDialog"
      />
      <Button
        label="开始解包"
        @click="startUnpack"
        :disabled="!options.OutputDir || (options.EnableDecrypt && !decryptKey)"
        autofocus
      />
    </div>
  </Dialog>

  <!-- Stage 2: Progress -->
  <Dialog
    v-else-if="currentStage === 'progress'"
    v-model:visible="visible"
    modal
    header="解包进行中"
    :style="{ width: '28rem' }"
    :closable="false"
  >
    <div class="mb-6" v-if="currentProgress">
      <div class="flex justify-between mb-2">
        <span class="text-gray-600">总进度 {{ Math.round(currentProgress.UnpackProgress) }}%</span>
        <span class="text-gray-500 text-sm">{{ currentProgress.UnpackCurrent }} / {{ currentProgress.UnpackTotal }}</span>
      </div>
      <ProgressBar :value="Math.round(currentProgress.UnpackProgress)" class="h-3"/>
    </div>

    <div
      class="bg-gray-50 rounded-lg p-4 mb-4"
      v-if="currentProgress?.UnpackCurrentFile"
    >
      <div class="text-sm text-gray-600 mb-1">当前文件</div>
      <div class="font-mono text-sm truncate">{{ currentProgress.UnpackCurrentFile }}</div>
    </div>

    <div class="text-center text-gray-500 text-sm" v-if="currentProgress">
      已处理 {{ currentProgress.UnpackCurrent }} 个文件，共 {{ currentProgress.UnpackTotal }} 个文件
    </div>

    <div class="flex justify-end">
      <Button
        label="后台运行"
        severity="secondary"
        @click="minimizeDialog"
      />
    </div>
  </Dialog>

  <!-- Stage 3: Completion -->
  <Dialog
    v-else-if="currentStage === 'complete'"
    v-model:visible="visible"
    modal
    header="解包完成"
    :style="{ width: '28rem' }"
  >
    <div class="text-center mb-6">
      <i class="pi pi-check-circle text-6xl text-green-500 mb-3"></i>
      <div class="text-xl font-semibold mb-2">解包成功完成</div>
      <div class="text-gray-600" v-if="currentProgress">
        已解包 {{ currentProgress.UnpackTotal }} 个文件
      </div>
    </div>

    <div class="bg-gray-50 rounded-lg p-4 mb-4" v-if="currentProgress?.UnpackSavePath">
      <div class="text-sm text-gray-600 mb-1">输出目录</div>
      <div class="font-mono text-sm truncate">{{ currentProgress.UnpackSavePath }}</div>
    </div>

    <div class="flex justify-end gap-2">
      <Button
        label="关闭"
        severity="secondary"
        @click="closeDialog"
      />
      <Button
        label="打开目录"
        icon="pi pi-folder"
        @click="openOutputDirectory"
        autofocus
      />
    </div>
  </Dialog>

  <!-- Stage 4: Error -->
  <Dialog
    v-else-if="currentStage === 'error'"
    v-model:visible="visible"
    modal
    header="解包失败"
    :style="{ width: '28rem' }"
  >
    <div class="text-center mb-6">
      <i class="pi pi-exclamation-circle text-6xl text-red-500 mb-3"></i>
      <div class="text-xl font-semibold mb-2">解包过程中出现错误</div>
      <div class="text-gray-600 text-sm mt-4" v-if="currentProgress?.UnpackErrorMessage">
        {{ currentProgress.UnpackErrorMessage }}
      </div>
    </div>

    <div class="bg-gray-50 rounded-lg p-4 mb-4" v-if="currentProgress?.UnpackSavePath">
      <div class="text-sm text-gray-600 mb-1">部分输出目录</div>
      <div class="font-mono text-sm truncate">{{ currentProgress.UnpackSavePath }}</div>
    </div>

    <div class="flex justify-end">
      <Button
        label="关闭"
        severity="secondary"
        @click="closeDialog"
      />
    </div>
  </Dialog>
</template>

<style scoped>
</style>