<!--suppress CssUnusedSymbol -->
<style scoped>

.show-terminal {
  position: relative;
  z-index: 11110;
  top: 1em;
  left: 1em;
  padding: 0.3em;
  background-color: #333;
  color: #fff;
  cursor: pointer;
}

.terminal {
  position: relative;
  z-index: 11111;
  top: 0;
  right: 0;
  width: 35em;
  max-width: 90%;
  height: 100%;
  min-height: 20vh;
  background-color: #030638;
  color: #ffff0a;
  border-radius: 8px;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.5);
  overflow: hidden;
  font-family: monospace, -moz-fixed;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

.terminal-header {
  background-color: #333;
  display: flex;
  justify-content: space-between; /* Ensures space between elements */
  align-items: center;
  padding: 10px;
  color: #fff;
}

.terminal-header .buttons {
  display: flex;
  gap: 8px;
  margin-left: auto; /* Pushes the buttons to the right */
  font-size: 0.8em;
  text-align: center;
}

.terminal-header .buttons span {
  display: block;
  width: 1.2em;
  height: 1.2em;
  border-radius: 50%;
}

.terminal-header .buttons .close {
  background-color: #ff5f56;
}

.terminal-header .buttons .close:hover {
  background-color: #f83e34;
}

.terminal-header .title {
  flex-grow: 1;
  text-align: center;
  font-size: 0.8em;
}

.terminal-body {
  padding: 15px;
  color: #fff;
  font-size: 14px;
  line-height: 1.6;
}


</style>
<template>
  <div class="show-terminal" @click="logVisible=true">
    <button title="Click to show log messages in this window">📎 show Log</button>
  </div>
  <div class="terminal" v-show="logVisible">
    <div class="terminal-header">
      <div class="title">📎 log {{ logTitle }} [{{ logMessagesCount }}]</div>
      <div class="buttons" @click="logVisible=false" title="Click to close the log window">
        <span class="close">X</span>
      </div>
    </div>
    <div class="terminal-body">
      <template v-for="(line, index) in myProps.msgLog ?? []" :key="index">
        <div>{{index}}: {{ line }}</div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { getLog } from "@/config";

const moduleName = "LogWindow";
const log = getLog(moduleName, 4, 2);
const logVisible = ref(true);
const logTitle = ref("📎 log window");
//// COMPONENT PROPERTIES
const myProps = defineProps<{
  titleLog: string
  msgLog?: string[] | undefined
}>();


//// EVENT SECTION

const emit = defineEmits(["log-clear", "log-hide", "log-show"]);

//// WATCH SECTION
watch(
  () => myProps.titleLog,
  (val, oldValue) => {
    // log.t(`🔎🔎 watch myProps.titleLog old: ${oldValue}, new val: ${val}`);
    if (val !== undefined) {
      if (val !== oldValue) {
        logTitle.value = val;
      }
    }
  },
  //  { immediate: true }
);
watch(
  () => myProps.msgLog,
  (val, oldValue) => {
     log.t(`🔎🔎 watch myProps.msgLog \noldValue: ${oldValue} \nnew val: ${val}`);
  },
  { deep: true },
);
//// COMPUTED SECTION
const logMessagesCount = computed(() => myProps.msgLog?.length ?? 0);
//// FUNCTIONS SECTION


onMounted(() => {
  log.t(`✅ ✅mounted ${moduleName} props.t:${myProps.titleLog}`);
  logTitle.value = myProps.titleLog;
});
</script>
