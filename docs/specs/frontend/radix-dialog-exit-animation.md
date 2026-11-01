# Radix 弹窗退场动画

## 现象

关闭任意 Dialog / AlertDialog / DropdownMenu / Tooltip 时，退出动画播放到最后一帧后，
会短暂闪回一次完整、可见的弹窗，然后才从 DOM 移除。

## 根因

- `tw-animate-css` 的 `animate-out` 默认 `animation-fill-mode: none`，动画结束后元素会
  立即恢复基础样式（`opacity: 1`、`scale: 1`）。
- Radix Presence 依赖 `animationend` 才卸载节点。它在事件处理里临时写入
  `style.animationFillMode = "forwards"`，随后用 `setTimeout(0)` 把内联值改回去。
- 在 WebKitGTK / React 18 下，这个内联重置可能先于 React 提交卸载发生。此时动画已经
  结束、`fill-mode` 又被清空，浏览器就会按基础样式绘制一帧完整弹窗，即“最后一帧闪烁”。

## 约束

**Don't**

- 不要只依赖 Radix 的内联 `animationFillMode` 兜底。
- 不要给任何 Radix Presence 的 `data-[state=closed]:animate-out` 元素省略
  `data-[state=closed]:fill-mode-forwards`。
- 不要只给内容层加退出动画而漏掉 Overlay；两层时长不一致会让遮罩先消失。

**Instead**

- 每个使用 `data-[state=closed]:animate-out` 的组件类名都必须同时包含
  `data-[state=closed]:fill-mode-forwards`，让样式表在动画结束后继续持有
  `opacity: 0` / 缩放后的最后一帧，直到 Radix 真正卸载节点。
- Dialog / AlertDialog 的 Overlay 与 Content 统一使用 `duration-md`，保证遮罩与内容
  按同一个用户动画倍率同步退场。
- 退出期间加 `data-[state=closed]:pointer-events-none`，避免动画中的弹窗继续接收点击。

当前覆盖位置：

- `frontend/src/components/ui/dialog.tsx`
- `frontend/src/components/ui/alert-dialog.tsx`
- `frontend/src/components/ui/dropdown-menu.tsx`
- `frontend/src/components/ui/tooltip.tsx`

## 验证

在无头服务器 + Playwright 中关闭弹窗，监听 `animationend`：

1. Radix 处理完后，计算样式应为 `animation-fill-mode: forwards`、`opacity: 0`。
2. 手动清空内联 `animationFillMode`（模拟 Radix 的 `setTimeout` 重置）后，计算样式仍须
   保持 `forwards` / `opacity: 0`。
3. 节点卸载前不得出现 `opacity: 1` 的采样帧。
