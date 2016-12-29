package display

import (
	"bytes"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/google/badwolf/triple/literal"
	"github.com/google/badwolf/triple/node"
	"github.com/wallix/awless/cloud/aws"
	"github.com/wallix/awless/rdf"
)

func FullDiff(diff *rdf.Diff, rootNode *node.Node) {
	table, err := tableFromDiff(diff, rootNode)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	table.Fprint(os.Stdout)
}

func ResourceDiff(diff *rdf.Diff, rootNode *node.Node) {
	diff.FullGraph().VisitDepthFirst(rootNode, func(g *rdf.Graph, n *node.Node, distance int) {
		var lit *literal.Literal
		diff, err := g.TriplesForSubjectPredicate(n, rdf.DiffPredicate)
		if len(diff) > 0 && err == nil {
			lit, _ = diff[0].Object().Literal()
		}

		var tabs bytes.Buffer
		for i := 0; i < distance; i++ {
			tabs.WriteByte('\t')
		}

		switch lit {
		case rdf.ExtraLiteral:
			color.Set(color.FgGreen)
			fmt.Fprintf(os.Stdout, "%s%s, %s\n", tabs.String(), n.Type(), n.ID())
			color.Unset()
		case rdf.MissingLiteral:
			color.Set(color.FgRed)
			fmt.Fprintf(os.Stdout, "%s%s, %s\n", tabs.String(), n.Type(), n.ID())
			color.Unset()
		default:
			fmt.Fprintf(os.Stdout, "%s%s, %s\n", tabs.String(), n.Type(), n.ID())
		}
	})
}

func tableFromDiff(diff *rdf.Diff, rootNode *node.Node) (*Table, error) {
	table := NewTable([]*PropertyDisplayer{
		{Property: "Type", DontTruncate: true},
		{Property: "Name/Id", DontTruncate: true},
		{Property: "Property", DontTruncate: true},
		{Property: "Value", DontTruncate: true},
	})
	table.MergeIdenticalCells = true

	diff.FullGraph().VisitDepthFirstUnique(rootNode, func(g *rdf.Graph, n *node.Node, distance int) {
		var lit *literal.Literal
		diffTriples, err := g.TriplesForSubjectPredicate(n, rdf.DiffPredicate)
		if len(diffTriples) > 0 && err == nil {
			lit, _ = diffTriples[0].Object().Literal()
		}

		var commonResource, changedProperties bool
		var displayF func(a ...interface{}) string

		switch lit {
		case rdf.ExtraLiteral:
			displayF = color.New(color.FgGreen).SprintFunc()
		case rdf.MissingLiteral:
			displayF = color.New(color.FgRed).SprintFunc()
		default:
			commonResource = true
			displayF = fmt.Sprint
		}
		if commonResource {
			changedProperties = addDiffProperties(table, g, n, diff)
		}
		if !commonResource || changedProperties {
			table.AddRow(displayF(n.Type()), displayF(n.ID()))
		}
	})

	table.SetSortBy("Type", "Name/Id", "Property", "Value")
	return table, nil
}

func addDiffProperties(table *Table, g *rdf.Graph, n *node.Node, diff *rdf.Diff) (hasChanges bool) {
	propertiesT, err := g.TriplesForSubjectPredicate(n, rdf.PropertyPredicate)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return false
	}

	for _, t := range propertiesT {
		if diff.HasInsertedTriple(t) {
			hasChanges = true
			prop, err := aws.NewPropertyFromTriple(t)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return hasChanges
			}
			displayF := color.New(color.FgGreen).SprintFunc()
			table.AddRow(fmt.Sprint(n.Type()), fmt.Sprint(n.ID()), displayF(prop.Key), displayF(prop.Value))
		}
		if diff.HasDeletedTriple(t) {
			hasChanges = true
			prop, err := aws.NewPropertyFromTriple(t)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				return hasChanges
			}

			displayF := color.New(color.FgRed).SprintFunc()
			table.AddRow(fmt.Sprint(n.Type()), fmt.Sprint(n.ID()), displayF(prop.Key), displayF(prop.Value))
		}
	}
	return hasChanges
}