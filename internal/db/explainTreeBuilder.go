package db

import (
	"fmt"
)

type QueryPlanNode struct {
	ID       string
	ParentID string
	NotUsed  string
	Detail   string
	Children []*QueryPlanNode
}

func BuildQueryPlanTree(rows [][]string) (*QueryPlanNode, error) {
	var nodes []*QueryPlanNode
	nodeMap := make(map[string]*QueryPlanNode)

	for _, row := range rows {
		id := row[0]
		parentId := row[1]
		notUsed := row[2]
		detail := row[3]

		node := &QueryPlanNode{
			ID:       id,
			ParentID: parentId,
			NotUsed:  notUsed,
			Detail:   detail,
		}

		nodes = append(nodes, node)
		nodeMap[id] = node
	}

	root := &QueryPlanNode{}
	for _, node := range nodes {
		if node.ParentID == "0" {
			root = node
		} else {
			parent := nodeMap[node.ParentID]
			parent.Children = append(parent.Children, node)
		}
	}

	return root, nil
}

func PrintQueryPlanTree(node *QueryPlanNode, indent string) {
	fmt.Printf("%s%s\n", indent, node.Detail)
	for _, child := range node.Children {
		PrintQueryPlanTree(child, indent+"  ")
	}
}
