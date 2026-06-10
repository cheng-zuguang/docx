package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func runInstallAgentHook(args []string, cwd string, stdout io.Writer) error {
	host := ""
	proposeMode := false
	for _, arg := range args {
		switch arg {
		case "--propose":
			proposeMode = true
		default:
			if strings.HasPrefix(arg, "--") {
				return fmt.Errorf("docx install-agent-hook: unknown option %q", arg)
			}
			if host != "" {
				return fmt.Errorf("docx install-agent-hook: expected codex or claude")
			}
			host = arg
		}
	}
	if host == "" {
		return fmt.Errorf("docx install-agent-hook: expected codex or claude")
	}
	root, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	if _, err := loadConfig(root); err != nil {
		return err
	}
	command := "docx finish"
	if proposeMode {
		command += " --propose"
	}
	switch host {
	case "codex":
		if err := installCodexAgentHook(root, command); err != nil {
			return err
		}
	case "claude":
		if err := installClaudeAgentHook(root, command); err != nil {
			return err
		}
	default:
		return fmt.Errorf("docx install-agent-hook: unsupported agent host %q", host)
	}
	fmt.Fprintf(stdout, "Installed docx %s agent hook\n", host)
	return nil
}

func installCodexAgentHook(root string, command string) error {
	configDir := filepath.Join(root, ".codex")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return upsertLifecycleHook(filepath.Join(configDir, "hooks.json"), "Stop", command)
}

func installClaudeAgentHook(root string, command string) error {
	configDir := filepath.Join(root, ".claude")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	return upsertLifecycleHook(filepath.Join(configDir, "settings.json"), "Stop", command)
}

func upsertLifecycleHook(path string, event string, command string) error {
	config, err := readHookConfig(path)
	if err != nil {
		return err
	}
	hooks := hookMap(config)
	eventHooks := hookMatcherList(hooks[event])
	if lifecycleHookHasCommand(eventHooks, command) {
		hooks[event] = eventHooks
		return writeJSON(path, config)
	}
	hooks[event] = append(eventHooks, map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": command,
			},
		},
	})
	return writeJSON(path, config)
}

func readHookConfig(path string) (map[string]interface{}, error) {
	config := map[string]interface{}{}
	bytes, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func hookMap(config map[string]interface{}) map[string]interface{} {
	hooks, ok := config["hooks"].(map[string]interface{})
	if !ok {
		hooks = map[string]interface{}{}
		config["hooks"] = hooks
	}
	return hooks
}

func hookMatcherList(value interface{}) []interface{} {
	items, ok := value.([]interface{})
	if !ok {
		return []interface{}{}
	}
	return items
}

func lifecycleHookHasCommand(matchers []interface{}, command string) bool {
	for _, item := range matchers {
		matcher, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for _, hookItem := range hookMatcherList(matcher["hooks"]) {
			hook, ok := hookItem.(map[string]interface{})
			if !ok {
				continue
			}
			if hookCommandMatches(hook, command) {
				return true
			}
		}
	}
	return false
}

func hookCommandMatches(hook map[string]interface{}, command string) bool {
	if hook["type"] != "command" {
		return false
	}
	hookCommand, _ := hook["command"].(string)
	if hookCommand == command {
		return true
	}
	if hookCommand != "docx" || !strings.HasPrefix(command, "docx ") {
		return false
	}
	args, ok := hook["args"].([]interface{})
	if !ok {
		return false
	}
	var parts []string
	for _, arg := range args {
		part, ok := arg.(string)
		if !ok {
			return false
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " ") == strings.TrimPrefix(command, "docx ")
}
