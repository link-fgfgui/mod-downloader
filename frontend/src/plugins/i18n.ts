import { createI18n } from "vue-i18n";

const messages = {
  zh: {
    welcome: "欢迎来到 Wails 桌面应用",
    about: "关于我们",
    changeLang: "切换语言",
    nav: {
      home: "首页",
      download: "下载",
      manage: "管理",
      unpin: "取消固定",
      settings: "设置",
    },
    search: {
      noMoreResults: "没有更多结果",
    },
    unpin: {
      title: "已固定的 Mod 版本",
      refresh: "刷新",
      unpin: "取消固定",
      unpinAll: "批量取消固定",
      empty: "当前没有已固定的 Mod 版本。",
      removed: "已取消固定",
      confirmAll: {
        title: "批量取消固定？",
        body: "将取消当前筛选范围内的全部固定版本，下次下载会重新使用最新版本。",
        confirm: "取消固定",
        cancel: "保留",
      },
      columns: {
        platform: "平台",
        modId: "Mod ID",
        versionId: "版本 ID",
        minecraftVersion: "Minecraft 版本",
        modLoader: "加载器",
      },
    },
    settings: {
      title: "设置",
      theme: { label: "主题", dark: "深色", light: "浅色", system: "跟随系统", saved: "主题已应用" },
      minecraftDir: { label: ".minecraft 目录", choose: "选择目录", valid: "目录有效", invalid: "目录无效或为空" },
      apiKeys: {
        curseforge: { label: "CurseForge API Key", status: "已设置", statusEmpty: "未设置", placeholder: "输入新的 API Key", save: "保存", clear: "清除", hint: "配置后可搜索 CurseForge 资源" },
        modrinth: { label: "Modrinth API Key", status: "已设置", statusEmpty: "未设置", placeholder: "输入新的 API Key", save: "保存", clear: "清除", hint: "当前未启用，保留供未来使用" },
        saved: "API Key 已保存",
        cleared: "API Key 已清除",
      },
    },
    manage: {
      title: "本地 Mod 管理",
      refresh: "刷新",
      noInstance: "请先在左侧选择一个已安装的实例。",
      noMods: "当前实例没有可解析的 Mod。",
      enabled: "启用",
      disabled: "禁用",
      copyNames: "复制名称",
      copyIds: "复制 ID",
      columns: {
        mod: "Mod",
        version: "版本",
        file: "文件",
        state: "状态",
      },
    },
    download: {
      selectDirTitle: "需要先选择有效的 .minecraft 目录",
      selectDirDesc: "当前目录为空或不可用，请先在左侧底部选择 .minecraft folder。",
      noMatchingVersions: "没有匹配当前 Minecraft 版本和 Mod Loader 的版本。",
      queued: "已加入下载队列",
      selection: {
        count: "已选 {n} 项",
        downloadAll: "全部下载",
        unpin: "取消固定",
        copyNames: "复制名称",
        deselectAll: "取消选择",
      },
      confirmReplace: {
        updateTitle: "更新到最新版本？",
        updateBody: "将把已安装的旧版本 jar 重命名为 .old 后缀，并下载该项目的最新版本。提示：右键按钮可跳过此确认。",
        conflictTitle: "替换为该项目的版本？",
        conflictBody: "已安装一个相同 mod ID 的文件（来自其他来源），将被重命名为 .old 后缀，并替换为该项目的最新版本。提示：右键按钮可跳过此确认。",
        confirm: "替换",
        cancel: "取消",
      },
      errors: {
        generic: "下载失败",
        failedWithReason: "{fileName} 下载失败：{reason}",
        invalidRequest: "请求无效",
        noMatchingVersion: "没有匹配选中实例的版本",
        missingDownloadUrl: "缺少下载链接",
        noSelectedVersion: "请先在左侧选择一个已安装的实例",
        unknownFile: "文件",
      },
    },
    aboutDescription:
      "这是一个基于 Wails + Vue3 + Vuetify + Router + i18n 的高配置桌面端模板。",
  },
  en: {
    welcome: "Welcome to Wails Desktop App",
    about: "About Us",
    changeLang: "Change Language",
    nav: {
      home: "Home",
      download: "Download",
      manage: "Manage",
      unpin: "Unpin",
      settings: "Settings",
    },
    search: {
      noMoreResults: "No more results",
    },
    unpin: {
      title: "Pinned Mod Versions",
      refresh: "Refresh",
      unpin: "Unpin",
      unpinAll: "Unpin All",
      empty: "No pinned mod versions yet.",
      removed: "Unpinned",
      confirmAll: {
        title: "Unpin all?",
        body: "This will unpin all versions in the current filtered range. Downloads will use the latest version next time.",
        confirm: "Unpin",
        cancel: "Keep",
      },
      columns: {
        platform: "Platform",
        modId: "Mod ID",
        versionId: "Version ID",
        minecraftVersion: "Minecraft Version",
        modLoader: "Mod Loader",
      },
    },
    settings: {
      title: "Settings",
      theme: { label: "Theme", dark: "Dark", light: "Light", system: "System", saved: "Theme applied" },
      minecraftDir: { label: ".minecraft Directory", choose: "Choose Directory", valid: "Directory valid", invalid: "Directory invalid or empty" },
      apiKeys: {
        curseforge: { label: "CurseForge API Key", status: "Set", statusEmpty: "Not set", placeholder: "Enter new API Key", save: "Save", clear: "Clear", hint: "Required to search CurseForge resources" },
        modrinth: { label: "Modrinth API Key", status: "Set", statusEmpty: "Not set", placeholder: "Enter new API Key", save: "Save", clear: "Clear", hint: "Not currently enabled; reserved for future use" },
        saved: "API Key saved",
        cleared: "API Key cleared",
      },
    },
    manage: {
      title: "Local Mod Management",
      refresh: "Refresh",
      noInstance: "Select an installed instance on the left first.",
      noMods: "No parseable mods were found in the selected instance.",
      enabled: "Enabled",
      disabled: "Disabled",
      copyNames: "Copy Names",
      copyIds: "Copy IDs",
      columns: {
        mod: "Mod",
        version: "Version",
        file: "File",
        state: "State",
      },
    },
    download: {
      selectDirTitle: "Please select a valid .minecraft directory first",
      selectDirDesc:
        "The current directory is empty or unavailable. Please select a .minecraft folder at the bottom-left first.",
      noMatchingVersions:
        "No versions match the current Minecraft version and Mod Loader.",
      queued: "Added to download queue",
      selection: {
        count: "{n} selected",
        downloadAll: "Download All",
        unpin: "Unpin",
        copyNames: "Copy Names",
        deselectAll: "Deselect All",
      },
      confirmReplace: {
        updateTitle: "Update to latest version?",
        updateBody: "The installed older jar will be renamed with a .old suffix and the project's latest version downloaded. Tip: right-click the button to skip this confirmation.",
        conflictTitle: "Replace with this project's version?",
        conflictBody: "An installed file with the same mod ID (from another source) will be renamed with a .old suffix and replaced with this project's latest version. Tip: right-click the button to skip this confirmation.",
        confirm: "Replace",
        cancel: "Cancel",
      },
      errors: {
        generic: "Download failed",
        failedWithReason: "{fileName} failed to download: {reason}",
        invalidRequest: "Invalid request",
        noMatchingVersion: "No version matches the selected instance",
        missingDownloadUrl: "Missing download URL",
        noSelectedVersion: "Select an installed instance on the left first",
        unknownFile: "File",
      },
    },
    aboutDescription:
      "A feature-rich desktop template based on Wails + Vue3 + Vuetify + Router + i18n.",
  },
};

const i18n = createI18n({
  legacy: false, // Set to false to use Composition API in Vue 3
  locale: "zh", // Default language
  fallbackLocale: "en",
  messages,
});

export default i18n;
