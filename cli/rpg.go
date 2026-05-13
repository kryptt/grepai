package cli

import (
	"fmt"
	"strings"

	"github.com/yoanbernabeu/grepai/rpg"
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
