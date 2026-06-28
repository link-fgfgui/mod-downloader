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
    },
    search: {
      noMoreResults: "没有更多结果",
    },
    manage: {
      title: "本地 Mod 管理",
      refresh: "刷新",
      noInstance: "请先在左侧选择一个已安装的实例。",
      noMods: "当前实例没有可解析的 Mod。",
      enabled: "启用",
      disabled: "禁用",
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
    },
    search: {
      noMoreResults: "No more results",
    },
    manage: {
      title: "Local Mod Management",
      refresh: "Refresh",
      noInstance: "Select an installed instance on the left first.",
      noMods: "No parseable mods were found in the selected instance.",
      enabled: "Enabled",
      disabled: "Disabled",
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
