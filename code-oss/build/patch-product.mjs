#!/usr/bin/env node
//
// Rewrite the build-time product.json so the inlined product object used by
// the workbench code paths that bypass IProductService cannot reach Microsoft
// or GitHub services.
//
// `IWorkbenchConstructionOptions.productConfiguration` only overrides
// IProductService; several consumers (chat setup, entitlement, setup providers)
// import the module-level product object instead. Patching product.json before
// the bundle is built is the only way to affect them.
//
// Usage: node patch-product.mjs <path-to-product.json>

import { readFile, writeFile } from 'node:fs/promises';

const DOCS_URL = 'https://github.com/vanduc2514/code-oss-container';

const REPO_URL_KEYS = [
	'documentationUrl',
	'termsStatementUrl',
	'privacyStatementUrl',
	'skusDocumentationUrl',
	'optimizeUsageDocumentationUrl',
	'publicCodeMatchesUrl',
	'managePlanUrl',
	'upgradePlanUrl',
	'signUpUrl'
];

const CLEARED_URL_KEYS = [
	'entitlementUrl',
	'entitlementSignupLimitedUrl',
	'tokenEntitlementUrl',
	'mcpRegistryDataUrl',
	'mcpConnectorsUrl',
	'managedSettingsUrl'
];

const file = process.argv[2];
if (!file) {
	console.error('usage: patch-product.mjs <path-to-product.json>');
	process.exit(1);
}

const product = JSON.parse(await readFile(file, 'utf8'));

const agent = product.defaultChatAgent;
if (agent && typeof agent === 'object') {
	for (const key of REPO_URL_KEYS) {
		if (key in agent) {
			agent[key] = DOCS_URL;
		}
	}
	for (const key of CLEARED_URL_KEYS) {
		if (key in agent) {
			agent[key] = '';
		}
	}
}

for (const key of ['voiceWsUrl', 'webviewContentExternalBaseUrlTemplate']) {
	if (key in product) {
		product[key] = '';
	}
}

await writeFile(file, `${JSON.stringify(product, null, '\t')}\n`);
