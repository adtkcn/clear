<script setup lang="ts">
import {ref, onMounted} from 'vue';
import {Events} from "@wailsio/runtime";
import {ClearService, ScanEvent} from "../bindings/changeme";

const disks = ref<string[]>([]);
const disk = ref('');
const dir = ref('');
const scanList = ref<string[]>([]);
const deleteList = ref<string[]>([]);
const scanning = ref(false);
const toast = ref({visible: false, error: false, msg: ''});
let toastTimer: ReturnType<typeof setTimeout>;

function showToast(message: string, isError = false) {
  toast.value = {visible: true, error: isError, msg: message};
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { toast.value.visible = false; }, 4000);
}

function checkDeleteStatus(path: string) {
  return deleteList.value.includes(path);
}

function deleteScanItem(index: number) {
  scanList.value.splice(index, 1);
}

async function scan() {
  const target = disk.value + dir.value;
  if (!target) {
    showToast('请输入路径或者选择磁盘', true);
    return;
  }
  scanList.value = [];
  deleteList.value = [];
  const msg = await ClearService.StartScan(target);
  showToast('已发送扫描指令：' + target);
  if (msg !== '执行中') {
    showToast(msg, true);
  }
}

async function sendDeleteDirs() {
  if (scanList.value.length === 0) {
    showToast('没有需要删除的目录', true);
    return;
  }
  const msg = await ClearService.DeleteDirs([...scanList.value]);
  showToast('已发送删除指令：' + msg);
}

onMounted(async () => {
  disks.value = (await ClearService.GetDisks()) || [];
  if (disks.value.length > 0) {
    disk.value = disks.value[0];
  }
  // 后端扫描/删除进度推送（原 websocket 的替代）
  Events.On('scanEvent', async (e: { data: ScanEvent }) => {
    const d = e.data;
    switch (d.type) {
      case 'ScanDirs':
        scanList.value.push(d.data);
        break;
      case 'DeleteDir':
        deleteList.value.push(d.data);
        break;
      case 'DeleteFailed':
        showToast('白名单目录，拒绝删除：' + d.data, true);
        break;
      case 'ScanDone':
        scanning.value = false;
        showToast(`扫描完成，共扫出 ${scanList.value.length} 个目录`);
        break;
    }
  });
});
</script>

<template>
  <main class="app">
    <header class="app-header">
      <div class="app-title"><span class="title-accent">x-</span>clear</div>
      <div class="app-subtitle">node_modules 目录扫描清理工具</div>
    </header>

    <div class="toolbar">
      <select class="disk-select" v-model="disk">
        <option value="" disabled>磁盘</option>
        <option v-for="d in disks" :key="d" :value="d">{{ d }}</option>
      </select>
      <input class="dir-input" v-model="dir" type="text" placeholder="扫描路径，如：Work\projects"/>
      <button class="btn" @click="scan">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        开始扫描
      </button>
    </div>

    <section class="card">
      <div class="card-header">
        <span>扫出的目录（{{ scanList.length }}）</span>
        <button class="btn btn-danger" @click="sendDeleteDirs">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/></svg>
          删除目录
        </button>
      </div>

      <div class="list">
        <div v-if="scanList.length === 0" class="list-empty">暂无数据，选择磁盘后点击“开始扫描”</div>
        <div v-for="(item, index) of scanList" :key="item" class="scan-item">
          <span class="scan-index">{{ index + 1 }}、</span>
          <span class="scan-path">{{ item }}</span>
          <span v-if="checkDeleteStatus(item)" class="state state-deleted" title="已删除">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          </span>
          <span v-else class="state state-remove" title="从列表中排除" @click="deleteScanItem(index)">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/></svg>
          </span>
        </div>
      </div>
    </section>

    <p class="tip">提示：删除前请检查一下，有些软件也会存在 node_modules 目录，以避免软件崩溃</p>
  </main>

  <div class="toast" :class="{'is-visible': toast.visible, 'is-error': toast.error}" role="status" aria-live="polite">
    <span class="toast-label">提示</span>
    <span class="toast-msg">{{ toast.msg }}</span>
  </div>
</template>
