package cursor

// cursorModelIDs maps "provider.alias" to a Cursor-native model id.
// Cursor subagents default to inheriting the parent model (model: inherit);
// entries here override only when an explicit Cursor model id is known.
var cursorModelIDs = map[string]string{}

// HostModelID resolves a provider/alias pair to a Cursor model id.
// When no mapping exists, returns ("inherit", false).
func HostModelID(provider, alias string) (string, bool) {
	if provider == "" && alias == "" {
		return "inherit", false
	}
	key := provider + "." + alias
	if id, ok := cursorModelIDs[key]; ok {
		return id, true
	}
	return "inherit", false
}
