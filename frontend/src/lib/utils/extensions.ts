// Extensions treated as code files (コードファイル拡張子セット)
export const CODE_EXTENSIONS: Set<string> = new Set([
	// Programming languages
	'.js',
	'.ts',
	'.jsx',
	'.tsx',
	'.py',
	'.rb',
	'.go',
	'.rs',
	'.java',
	'.kt',
	'.scala',
	'.cs',
	'.cpp',
	'.c',
	'.h',
	'.hpp',
	'.swift',
	'.m',
	'.php',
	'.lua',
	'.r',
	'.dart',
	'.ex',
	'.exs',
	'.zig',
	'.nim',
	'.v',
	'.pl',
	'.pm',
	'.clj',
	'.cljs',
	'.erl',
	'.hs',
	'.ml',
	'.fs',
	'.jl',

	// Web / Markup / Styles
	'.html',
	'.css',
	'.scss',
	'.sass',
	'.less',
	'.vue',
	'.svelte',
	'.astro',

	// Build systems
	'.rake',
	'.gradle',
	'.cmake',
	'.makefile',
	'.mk',

	// IaC / Infra (excluding config files)
	'.tf',
	'.tfvars',
	'.hcl',
	'.jsonnet',
	'.dhall',
	'.nix',
	'.pp',
	'.erb',

	// Data management
	'.sql',
	'.prisma',
	'.graphql',
	'.gql',
	'.proto',

	// Shell / Scripts
	'.sh',
	'.bash',
	'.zsh',
	'.ps1',
	'.bat',
]);

// Extensions treated as config files (設定ファイル拡張子セット)
export const CONFIG_EXTENSIONS: Set<string> = new Set([
	'.yaml',
	'.yml',
	'.json',
	'.toml',
	'.ini',
	'.cfg',
	'.conf',
	'.env',
	'.properties',
	'.xml',
	'.plist',
]);

// Check if extension is a code file (コードファイル判定)
export function isCodeExtension(ext: string): boolean {
	return CODE_EXTENSIONS.has(ext.toLowerCase());
}

// Check if extension is a config file (設定ファイル判定)
export function isConfigExtension(ext: string): boolean {
	return CONFIG_EXTENSIONS.has(ext.toLowerCase());
}
