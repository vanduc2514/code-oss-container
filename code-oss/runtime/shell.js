// Helpers shared by the two entry-point modules (index.js and agents.js).
// Kept in its own file so both pages can run under a `script-src 'self'`
// policy without inline scripts.

export function createAsUri(URI) {
	const base = new URL('.', document.location.href);
	return path => URI.parse(new URL(path, base).toString());
}

// Relative paths stay on this origin, so the browser never resolves a private
// agent-host address; nginx proxies it. Only these fields reach the workbench,
// which keeps proxy-only fields such as `upstream` out of the settings.
function toAddress(host) {
	const raw = host.address || host.path;
	if (typeof raw !== 'string' || raw === '') {
		return undefined;
	}
	try {
		const relative = raw.startsWith('/');
		const url = new URL(raw, document.location.href);
		if (relative) {
			url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
		} else if (url.protocol !== 'ws:' && url.protocol !== 'wss:') {
			return undefined;
		}
		return url.toString();
	} catch {
		return undefined;
	}
}

export async function readAgentHosts() {
	try {
		const response = await fetch('./config.json', { cache: 'no-store' });
		if (!response.ok) {
			return [];
		}
		const config = await response.json();
		const hosts = Array.isArray(config) ? config : Array.isArray(config.remoteAgentHosts) ? config.remoteAgentHosts : [];
		return hosts
			.filter(host => host && typeof host.name === 'string')
			.map(host => {
				const address = toAddress(host);
				if (!address) {
					return undefined;
				}
				const entry = { name: host.name, address };
				if (typeof host.connectionToken === 'string' && host.connectionToken !== '') {
					entry.connectionToken = host.connectionToken;
				}
				return entry;
			})
			.filter(Boolean);
	} catch {
		return [];
	}
}

// Upstream stores web secrets in localStorage with an optional server key;
// a static host has no server key, so plain localStorage matches the default.
export function createSecretStorage(key) {
	const read = () => {
		try {
			return JSON.parse(localStorage.getItem(key) || '{}');
		} catch {
			return {};
		}
	};
	const write = data => localStorage.setItem(key, JSON.stringify(data));
	return {
		type: 'persisted',
		async get(name) { return read()[name]; },
		async set(name, value) { const data = read(); data[name] = value; write(data); },
		async delete(name) { const data = read(); delete data[name]; write(data); },
		async keys() { return Object.keys(read()); }
	};
}
