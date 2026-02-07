// adapter-static (SPA) ビルド時は SSR を無効化
// adapter-node (Cloud Run) ビルド時はデフォルトの SSR を使用
export const ssr = !import.meta.env.VITE_SPA_MODE;
export const prerender = false;
