package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// printNumberOfRules collects and prints the number of rules handled by snapd.
func printNumberOfRules() error {
	d, err := os.ReadFile("/var/lib/snapd/interfaces-requests/request-rules.json")
	if err != nil {
		return err
	}

	type rule struct {
		Snap string
	}

	allRules := struct {
		Rules []rule
	}{}

	err = json.Unmarshal(d, &allRules)
	if err != nil {
		return err
	}

	stats := make(map[string]int)
	for _, r := range allRules.Rules {
		stats[r.Snap] = stats[r.Snap] + 1
	}

	keys := make([]string, 0, len(stats))
	var maxKeyLen int
	for k := range stats {
		keys = append(keys, k)
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}
	sort.Strings(keys)
	keys = append(keys, "total")
	stats["total"] = len(allRules.Rules)

	for _, k := range keys {
		fmt.Printf("%-*s %10d\n", maxKeyLen, k, stats[k])
	}

	return nil
}
