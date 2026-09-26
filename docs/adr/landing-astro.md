# 官网用 Astro 构建成静态站

状态：已接受

## 背景

`landing/` 原来是一份手写的 `index.html`，样式、中英文文案和 WebGL 脚本都内联在一个文件里，图片放在 `landing/public/`。页面变长以后，单文件难以按区块维护，也没有构建步骤可以接进发布流程。

## 决定

`landing/` 改为 Astro 项目，用 pnpm 管理，`pnpm --dir landing run build` 输出纯静态的 `landing/dist/`。

- 页面拆成 `src/layouts/Base.astro`（head、全局样式、脚本）和 `src/components/` 下按区块划分的组件，`src/pages/index.astro` 负责拼装。
- `<style>` 和 `<script>` 用 `is:inline`，原样进入输出，不经 Astro 作用域化或打包。
- `publicDir` 指向 `landing/static/`，图片仍以 `public/*.webp`、图标仍以 `favicon.svg` 的相对路径发布，页面引用不变。
- 关闭 `compressHTML`，保留原有空白。

### 中英文分开的地址

用 Astro 的 i18n 路由：中文是默认语言，在根路径 `/`；英文在 `/en/`（`prefixDefaultLocale: false`）。两页在构建时各自渲染出完整文案，HTML 里直接带着对应语言的文字，不再靠脚本在浏览器里换语言。文案集中在 `src/i18n/copy.ts`，中英两份字段必须一致。

- 每页有自己的 `<html lang>`、`<title>`、`description`、`canonical`，以及 `hreflang` 为 `zh`、`en` 和 `x-default`（指向中文根路径）的互链。站点地址固定为 `https://akproxy.akr.moe`。
- `@astrojs/sitemap` 生成带 `hreflang` 的站点地图，`robots.txt` 指向它。
- 语言切换是指向另一语言地址的普通链接。不根据浏览器语言自动跳转，也不记住上次的选择，避免搜索引擎和访客拿到与地址不符的内容。
- 页面里只剩下载按钮识别系统、点击复制地址和 WebGL 三段脚本。图片和图标改用根路径引用（`/public/…`、`/favicon.svg`），两个语言地址都能取到。

### 部署

官网部署在 Netlify 的 `akproxy` 站点，仓库根目录的 `netlify.toml` 指定在 `landing/` 里执行 `pnpm run build`，发布 `landing/dist`（Netlify 按仓库根目录解析 `publish`）。`landing/package.json` 用 `packageManager` 固定 pnpm 版本。`akproxy.akr.moe` 在 Cloudflare 上是指向 `akproxy.netlify.app` 的 CNAME，不开代理，证书由 Netlify 签发；它优先于 `*.akr.moe` 通配记录。站点关闭了 Netlify 的 “built with” 徽标。

分享卡片 `static/public/og-zh.jpg`、`og-en.jpg`（1200×630）由 `landing/og/og.html` 渲染截图得到，改动标题或配图时一起重新生成。

## 备选与后果

继续维护单个 HTML 最简单，但区块多了以后修改容易互相影响。改用 Astro 的作用域样式和脚本打包会重写选择器、拆出外部文件，迁移后的页面就不能保证和原来一致。

迁移时以构建产物对比原页面：解析后的标签、属性和文本序列一致；1440 和 390 宽度下中英文整页截图逐像素一致。拆成两个语言地址后，再次确认两页不执行脚本时的静态文字完整且只含本语言，渲染后的文字和整页截图都与原页面对应语言一致。之后改页面仍在 `src/` 下改，`dist/` 不提交。
