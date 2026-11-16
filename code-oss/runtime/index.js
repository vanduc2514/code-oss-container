import { createAsUri, createSecretStorage, readAgentHosts } from './shell.js';
import { commit, version } from './version.js';

const docsUrl = 'https://github.com/vanduc2514/code-oss-container';

// The workbench resolves bundles, workers and webview resources against this
// global, so it has to be set before the entry point module is evaluated.
globalThis._VSCODE_FILE_ROOT = new URL('./out/', document.location.href).toString();

const { create, URI } = await import('./out/vs/workbench/workbench.web.main.internal.js');

const asUri = createAsUri(URI);
const remoteAgentHosts = await readAgentHosts();

create(document.body, {
	workspaceProvider: {
		trusted: true,
		workspace: undefined,
		open: async () => false
	},
	additionalBuiltinExtensions: [
		asUri('./extensions/theme-defaults'),
		asUri('./extensions/theme-seti')
	],
	secretStorageProvider: createSecretStorage('code-oss-container.secrets'),
	settingsSyncOptions: {
		enabled: false
	},
	productConfiguration: {
		nameShort: 'Code - OSS',
		nameLong: 'Code - OSS',
		applicationName: 'code-oss',
		dataFolderName: '.vscode-oss',
		sharedDataFolderName: '.vscode-oss-shared',
		version,
		commit,
		quality: 'stable',
		enableTelemetry: false,
		licenseName: 'MIT',
		licenseUrl: `https://github.com/microsoft/vscode/blob/${version}/LICENSE.txt`,
		serverLicenseUrl: `https://github.com/microsoft/vscode/blob/${version}/LICENSE.txt`,
		documentationUrl: 'https://github.com/vanduc2514/code-oss-container',
		releaseNotesUrl: 'https://github.com/vanduc2514/code-oss-container',
		reportIssueUrl: 'https://github.com/vanduc2514/code-oss-container/issues',
		requestFeatureUrl: 'https://github.com/vanduc2514/code-oss-container/issues',
		keyboardShortcutsUrlMac: '',
		keyboardShortcutsUrlLinux: '',
		keyboardShortcutsUrlWin: '',
		introductoryVideosUrl: '',
		tipsAndTricksUrl: '',
		newsletterSignupUrl: '',
		youTubeUrl: '',
		urlProtocol: 'code-oss',
		voiceWsUrl: '',
		webviewContentExternalBaseUrlTemplate: asUri('./out/vs/workbench/contrib/webview/browser/pre/').toString(),
		extensionsGallery: {
			serviceUrl: 'https://open-vsx.org/vscode/gallery',
			itemUrl: 'https://open-vsx.org/vscode/item',
			resourceUrlTemplate: 'https://open-vsx.org/vscode/unpkg/{publisher}/{name}/{version}/{path}'
		},
		defaultChatAgent: {
			extensionId: 'GitHub.copilot',
			chatExtensionId: 'GitHub.copilot-chat',
			chatExtensionOutputId: 'GitHub.copilot-chat.GitHub Copilot Chat.log',
			chatExtensionOutputExtensionStateCommand: 'github.copilot.debug.extensionState',
			documentationUrl: docsUrl,
			skusDocumentationUrl: docsUrl,
			optimizeUsageDocumentationUrl: docsUrl,
			publicCodeMatchesUrl: docsUrl,
			managePlanUrl: docsUrl,
			upgradePlanUrl: docsUrl,
			signUpUrl: docsUrl,
			termsStatementUrl: docsUrl,
			privacyStatementUrl: docsUrl,
			provider: {
				default: { id: 'github', name: 'GitHub' },
				enterprise: { id: 'github-enterprise', name: 'GHE' },
				google: { id: 'google', name: 'Google' },
				apple: { id: 'apple', name: 'Apple' },
				microsoft: { id: 'microsoft', name: 'Microsoft' }
			},
			providerExtensionId: 'vscode.github-authentication',
			providerUriSetting: 'github-enterprise.uri',
			providerScopes: [],
			entitlementUrl: '',
			entitlementSignupLimitedUrl: '',
			tokenEntitlementUrl: '',
			mcpRegistryDataUrl: '',
			mcpConnectorsUrl: '',
			managedSettingsUrl: '',
			chatQuotaExceededContext: 'github.copilot.chat.quotaExceeded',
			completionsQuotaExceededContext: 'github.copilot.completions.quotaExceeded',
			walkthroughCommand: 'github.copilot.open.walkthrough',
			completionsMenuCommand: 'github.copilot.toggleStatusMenu',
			chatRefreshTokenCommand: 'github.copilot.refreshToken',
			generateCommitMessageCommand: 'github.copilot.git.generateCommitMessage',
			resolveMergeConflictsCommand: 'github.copilot.git.resolveMergeConflicts',
			completionsAdvancedSetting: 'github.copilot.advanced',
			completionsEnablementSetting: 'github.copilot.enable',
			nextEditSuggestionsSetting: 'github.copilot.nextEditSuggestions.enabled'
		}
	},
	configurationDefaults: {
		'workbench.colorTheme': 'Dark 2026',
		'workbench.startupEditor': 'none',
		'chat.remoteAgentHostsEnabled': true,
		'chat.remoteAgentHosts': remoteAgentHosts,
		'telemetry.telemetryLevel': 'off',
		'chat.allowAnonymousAccess': true
	}
});
