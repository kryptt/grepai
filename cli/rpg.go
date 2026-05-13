package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/alpkeskin/gotoon"
	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/rpg"
)

var (
	rpgJSON             bool
	rpgTOON             bool
	rpgSearchScope      string
	rpgSearchKinds      string
	rpgSearchLimit      int
	rpgExploreDirection string
	rpgExploreDepth     int
	rpgExploreEdgeTypes string
	rpgExploreLimit     int
)

func parseRPGNodeKinds(kindsStr string) ([]rpg.NodeKind, error) {
	if strings.TrimSpace(kindsStr) == "" {
		return nil, nil
	}
	parts := strings.Split(kindsStr, ",")
	kinds := make([]rpg.NodeKind, 0, len(parts))
	for _, part := range parts {
		kind := strings.TrimSpace(part)
		switch kind {
		case "area":
			kinds = append(kinds, rpg.KindArea)
		case "category":
			kinds = append(kinds, rpg.KindCategory)
		case "subcategory":
			kinds = append(kinds, rpg.KindSubcategory)
		case "file":
			kinds = append(kinds, rpg.KindFile)
		case "symbol":
			kinds = append(kinds, rpg.KindSymbol)
		case "chunk":
			kinds = append(kinds, rpg.KindChunk)
		case "":
			continue
		default:
			return nil, fmt.Errorf("invalid kind: %s", kind)
		}
	}
	return kinds, nil
}

func parseRPGEdgeTypes(edgeTypesStr string) ([]rpg.EdgeType, error) {
	if strings.TrimSpace(edgeTypesStr) == "" {
		return nil, nil
	}
	parts := strings.Split(edgeTypesStr, ",")
	edgeTypes := make([]rpg.EdgeType, 0, len(parts))
	for _, part := range parts {
		edgeType := strings.TrimSpace(part)
		switch edgeType {
		case "feature_parent":
			edgeTypes = append(edgeTypes, rpg.EdgeFeatureParent)
		case "contains":
			edgeTypes = append(edgeTypes, rpg.EdgeContains)
		case "invokes":
			edgeTypes = append(edgeTypes, rpg.EdgeInvokes)
		case "imports":
			edgeTypes = append(edgeTypes, rpg.EdgeImports)
		case "maps_to_chunk":
			edgeTypes = append(edgeTypes, rpg.EdgeMapsToChunk)
		case "semantic_sim":
			edgeTypes = append(edgeTypes, rpg.EdgeSemanticSim)
		case "":
			continue
		default:
			return nil, fmt.Errorf("invalid edge type: %s", edgeType)
		}
	}
	return edgeTypes, nil
}

func validateRPGDirection(direction string) error {
	if direction != "forward" && direction != "reverse" && direction != "both" {
		return fmt.Errorf("direction must be 'forward', 'reverse', or 'both'")
	}
	return nil
}

func loadLocalRPG(ctx context.Context) (*rpg.GOBRPGStore, *rpg.QueryEngine, error) {
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if !cfg.RPG.Enabled {
		return nil, nil, fmt.Errorf("RPG is not enabled or index is empty")
	}

	store := rpg.NewGOBRPGStore(config.GetRPGIndexPath(projectRoot))
	if err := store.Load(ctx); errors.Is(err, rpg.ErrRPGIndexOutdated) {
		return nil, nil, fmt.Errorf("RPG index is outdated; run 'grepai watch' to rebuild")
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to load RPG: %w", err)
	}
	stats, err := store.GetStats(ctx)
	if err != nil {
		_ = store.Close()
		return nil, nil, fmt.Errorf("failed to read RPG stats: %w", err)
	}
	if stats.TotalNodes == 0 {
		_ = store.Close()
		return nil, nil, fmt.Errorf("RPG is not enabled or index is empty")
	}

	return store, rpg.NewQueryEngine(store.GetGraph()), nil
}

func outputRPGStructured(data any) error {
	if rpgJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	if rpgTOON {
		output, err := gotoon.Encode(data)
		if err != nil {
			return fmt.Errorf("failed to encode TOON: %w", err)
		}
		fmt.Println(output)
	}
	return nil
}

func runRPGSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	kinds, err := parseRPGNodeKinds(rpgSearchKinds)
	if err != nil {
		return err
	}
	store, qe, err := loadLocalRPG(ctx)
	if err != nil {
		return err
	}
	defer store.Close()

	results, err := qe.SearchNode(ctx, rpg.SearchNodeRequest{
		Query: args[0],
		Scope: rpgSearchScope,
		Kinds: kinds,
		Limit: rpgSearchLimit,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	if rpgJSON || rpgTOON {
		return outputRPGStructured(results)
	}
	return displayRPGSearchResults(results)
}

func displayRPGSearchResults(results []rpg.SearchNodeResult) error {
	fmt.Printf("RPG Results (%d):\n", len(results))
	fmt.Println(strings.Repeat("-", 60))
	if len(results) == 0 {
		fmt.Println("No RPG nodes found.")
		return nil
	}
	for i, result := range results {
		node := result.Node
		name := node.Feature
		if node.SymbolName != "" {
			name = node.SymbolName
		}
		fmt.Printf("\n%d. %s (%s) score %.3f\n", i+1, name, node.Kind, result.Score)
		fmt.Printf("   Node: %s\n", node.ID)
		if result.FeaturePath != "" {
			fmt.Printf("   Feature path: %s\n", result.FeaturePath)
		}
		if node.Path != "" {
			fmt.Printf("   Path: %s", node.Path)
			if node.StartLine > 0 {
				fmt.Printf(":%d", node.StartLine)
			}
			fmt.Println()
		}
	}
	return nil
}
