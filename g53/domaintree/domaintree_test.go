package domaintree

import (
	"fmt"
	"testing"

	ut "cement/unittest"
	"g53"
)

/* The initial structure of rbtree
 *
 *             b
 *           /   \
 *          a    d.e.f
 *              /  |   \
 *             c   |    g.h
 *                 |     |
 *                w.y    i
 *              /  |  \
 *             x   |   z
 *                 |   |
 *                 p   j
 *               /   \
 *              o     q
 */
func treeInsertString(tree *DomainTree, n string) (*Node, error) {
	return tree.Insert(g53.NameFromStringUnsafe(n))
}

func createDomainTree(returnEmptyNode bool) *DomainTree {
	domains := []string{
		"c", "b", "a", "x.d.e.f", "z.d.e.f", "g.h", "i.g.h", "o.w.y.d.e.f",
		"j.z.d.e.f", "p.w.y.d.e.f", "q.w.y.d.e.f"}

	tree := NewDomainTree(returnEmptyNode)
	for i, d := range domains {
		node, _ := treeInsertString(tree, d)
		node.data = i + 1
	}
	return tree
}

func TestTreeNodeCount(t *testing.T) {
	ut.Equal(t, createDomainTree(false).NodeCount(), 14)
}

func TestTreeInsert(t *testing.T) {
	tree := createDomainTree(false)
	node, err := treeInsertString(tree, "c")
	ut.Equal(t, err, ErrAlreadyExist)

	node, err = treeInsertString(tree, "d.e.f")
	ut.Equal(t, err, nil)
	ut.Equal(t, node.name.String(true), "d.e.f")
	ut.Equal(t, tree.nodeCount, 14)

	node, err = tree.Insert(g53.Root)
	ut.Assert(t, err == nil, "inert root domain should ok but get %v", err)
	ut.Assert(t, g53.Root.Equals(node.name), "insert return node name should equals to insert name")
	ut.Equal(t, tree.nodeCount, 14)

	node, err = treeInsertString(tree, "example.com")
	ut.Assert(t, err == nil, "inert new domain should ok but get %v", err)
	ut.Equal(t, tree.nodeCount, 15)
	node.data = 12

	node, err = treeInsertString(tree, "example.com")
	ut.Equal(t, err, ErrAlreadyExist)
	ut.Equal(t, node.name.String(true), "example.com")
	ut.Equal(t, tree.nodeCount, 15)

	// split the node "d.e.f"
	node, err = treeInsertString(tree, "k.e.f")
	ut.Equal(t, node.name.String(true), "k")
	ut.Equal(t, tree.nodeCount, 17)

	// split the node "g.h"
	node, err = treeInsertString(tree, "h")
	ut.Equal(t, err, nil)
	ut.Equal(t, node.name.String(true), "h")
	ut.Equal(t, tree.nodeCount, 18)

	// add child domain
	node, err = treeInsertString(tree, "m.p.w.y.d.e.f")
	ut.Equal(t, node.name.String(true), "m")
	ut.Equal(t, tree.nodeCount, 19)

	node, err = treeInsertString(tree, "n.p.w.y.d.e.f")
	ut.Assert(t, err == nil, "insert new child name should ok but get %v", err)
	ut.Equal(t, node.name.String(true), "n")
	ut.Equal(t, tree.nodeCount, 20)

	node, err = treeInsertString(tree, "l.a")
	ut.Equal(t, node.name.String(true), "l")
	ut.Equal(t, tree.nodeCount, 21)

	_, err = treeInsertString(tree, "r.d.e.f")
	ut.Assert(t, err == nil, "insert new child name should ok but get %v", err)
	_, err = treeInsertString(tree, "s.d.e.f")
	ut.Assert(t, err == nil, "insert new child name should ok but get %v", err)
	ut.Equal(t, tree.nodeCount, 23)
	_, err = treeInsertString(tree, "h.w.y.d.e.f")
	ut.Assert(t, err == nil, "insert new child name should ok but get %v", err)

	node, err = treeInsertString(tree, "f")
	ut.Assert(t, err == nil, "f node has no data")
	node.SetData(1000)
	_, err = treeInsertString(tree, "f")
	ut.Assert(t, err == ErrAlreadyExist, "insert already exists domain should get error")

	newNames := []string{"m", "nm", "om", "k", "l", "fe", "ge", "i", "ae", "n"}
	for _, newName := range newNames {
		_, err = treeInsertString(tree, newName)
		ut.Assert(t, err == nil, "insert new child name should ok but get %v", err)
	}
}

func TestTreeSearch(t *testing.T) {
	tree := createDomainTree(false)
	node, ret := tree.Search(g53.NameFromStringUnsafe("a"))
	ut.Equal(t, ret, ExactMatch)
	ut.Equal(t, node.name.String(true), "a")

	notExistsNames := []string{
		"d.e.f", "y.d.e.f", "x", "m.n",
	}
	for _, n := range notExistsNames {
		_, ret := tree.Search(g53.NameFromStringUnsafe(n))
		ut.Equal(t, ret, NotFound)
	}

	tree = createDomainTree(true)
	exactMatchNames := []string{
		"d.e.f", "w.y.d.e.f",
	}
	for _, n := range exactMatchNames {
		_, ret := tree.Search(g53.NameFromStringUnsafe(n))
		ut.Equal(t, ret, ExactMatch)
	}

	// partial match
	node, ret = tree.Search(g53.NameFromStringUnsafe("m.b"))
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, node.name.String(true), "b")
	node, ret = tree.Search(g53.NameFromStringUnsafe("m.d.e.f"))
	ut.Equal(t, ret, PartialMatch)

	// find rbtnode
	node, ret = tree.Search(g53.NameFromStringUnsafe("q.w.y.d.e.f"))
	ut.Equal(t, ret, ExactMatch)
	ut.Equal(t, node.name.String(true), "q")
}

func TestTreeFlag(t *testing.T) {
	tree := createDomainTree(false)
	node, ret := treeInsertString(tree, "flags.example")
	ut.Equal(t, ret, nil)
	ut.Assert(t, node.GetFlag(NF_CALLBACK) == false, "by default, node has no flag")
	node.SetFlag(NF_CALLBACK, true)
	ut.Assert(t, node.GetFlag(NF_CALLBACK) == true, "node should has flag after set")
	node.SetFlag(NF_CALLBACK, false)
	ut.Assert(t, node.GetFlag(NF_CALLBACK) == false, "node should has no flag after reset")
}

func testCallback(node *Node, callbackChecker interface{}) bool {
	*(callbackChecker.(*bool)) = true
	return false
}

func TestTreeNodeCallback(t *testing.T) {
	tree := createDomainTree(false)
	node, err := treeInsertString(tree, "callback.example")
	ut.Equal(t, err, nil)
	node.data = 1
	ut.Assert(t, node.GetFlag(NF_CALLBACK) == false, "by default, node has no flag")
	node.SetFlag(NF_CALLBACK, true)
	// add more levels below and above the callback node for partial match.

	subNode, err := treeInsertString(tree, "sub.callback.example")
	ut.Equal(t, err, nil)
	node.data = 2
	parentNode, _ := treeInsertString(tree, "example")
	node, ret := tree.Search(g53.NameFromStringUnsafe("callback.example"))
	ut.Assert(t, node.GetFlag(NF_CALLBACK) == true, "node has set flag")
	ut.Assert(t, subNode.GetFlag(NF_CALLBACK) == false, "node hasn't set flag")
	ut.Assert(t, parentNode.GetFlag(NF_CALLBACK) == false, "node hasn't  set flag")

	// check if the callback is called from find()
	nodePath := NewNodeChain()
	callbackCalled := false
	node, ret = tree.SearchExt(g53.NameFromStringUnsafe("sub.callback.example"), nodePath, testCallback, &callbackCalled)
	ut.Equal(t, callbackCalled, true)

	// enable callback at the parent node, but it doesn't have data so
	// the callback shouldn't be called.
	nodePath2 := NewNodeChain()
	parentNode.SetFlag(NF_CALLBACK, true)
	callbackCalled = false
	node, ret = tree.SearchExt(g53.NameFromStringUnsafe("callback.example"), nodePath2, testCallback, &callbackCalled)
	ut.Equal(t, ret, ExactMatch)
	ut.Equal(t, callbackCalled, false)
}

func TestTreeNodeChain(t *testing.T) {
	chain := NewNodeChain()
	ut.Equal(t, chain.GetLevelCount(), 0)

	tree := NewDomainTree(true)
	treeInsertString(tree, ".")
	_, ret := tree.SearchExt(g53.NameFromStringUnsafe("."), chain, nil, nil)
	ut.Equal(t, ret, ExactMatch)
	ut.Equal(t, chain.GetLevelCount(), 1)

	/*
	 * Now creating a possibly deepest tree with MAX_LABELS levels.
	 * it should look like:
	 *           (.)
	 *            |
	 *            a
	 *            |
	 *            a
	 *            : (MAX_LABELS - 1) "a"'s
	 *
	 * then confirm that find() for the deepest name succeeds without any
	 * disruption, and the resulting chain has the expected level.
	 * Note that the root name (".") solely belongs to a single level,
	 * so the levels begin with 2.
	 */
	nodeName := g53.Root
	for i := 2; i <= g53.MAX_LABELS; i++ {
		nodeName, _ = g53.NameFromStringUnsafe("a").Concat(nodeName)
		_, err := tree.Insert(nodeName)
		ut.Equal(t, err, nil)

		chain := NewNodeChain()
		_, ret := tree.SearchExt(nodeName, chain, nil, nil)
		ut.Equal(t, ret, ExactMatch)
		ut.Equal(t, chain.GetLevelCount(), i)
	}
}

//
//the domain order should be:
// a, b, c, d.e.f, x.d.e.f, w.y.d.e.f, o.w.y.d.e.f, p.w.y.d.e.f, q.w.y.d.e.f,
// z.d.e.f, j.z.d.e.f, g.h, i.g.h
//             b
//           /   \
//          a    d.e.f
//              /  |   \
//             c   |    g.h
//                 |     |
//                w.y    i
//              /  |  \
//             x   |   z
//                 |   |
//                 p   j
//               /   \
//              o     q
///
func TestTreeNextNode(t *testing.T) {
	names := []string{
		"a", "b", "c", "d.e.f", "x.d.e.f", "w.y.d.e.f", "o.w.y.d.e.f",
		"p.w.y.d.e.f", "q.w.y.d.e.f", "z.d.e.f", "j.z.d.e.f", "g.h", "i.g.h"}
	tree := createDomainTree(false)
	nodePath := NewNodeChain()
	node, ret := tree.SearchExt(g53.NameFromStringUnsafe(names[0]), nodePath, nil, nil)
	ut.Equal(t, ret, ExactMatch)
	for i := 0; i < len(names); i++ {
		ut.Assert(t, node != nil, "node shouldn't be nil")
		ut.Equal(t, names[i], nodePath.GetAbsoluteName().String(true))
		node = tree.nextNode(nodePath)
	}

	// We should have reached the end of the tree.
	ut.Assert(t, node == nil, "node will reach the end")
}

func TestNonTerminal(t *testing.T) {
	tree := createDomainTree(true)
	node, _ := tree.Search(g53.NameFromStringUnsafe("c"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == false, "")

	node, _ = tree.Search(g53.NameFromStringUnsafe("d.e.f"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == true, "")

	node, _ = tree.Search(g53.NameFromStringUnsafe("w.y.d.e.f"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == true, "")

	node, _ = tree.Search(g53.NameFromStringUnsafe("p.w.y.d.e.f"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == false, "")

	node.SetData(nil)
	node, _ = tree.Search(g53.NameFromStringUnsafe("o.w.y.d.e.f"))
	node.SetData(nil)
	node, _ = tree.Search(g53.NameFromStringUnsafe("q.w.y.d.e.f"))
	node.SetData(nil)
	node, _ = tree.Search(g53.NameFromStringUnsafe("w.y.d.e.f"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == false, "")
	node, _ = tree.Search(g53.NameFromStringUnsafe("d.e.f"))
	ut.Assert(t, tree.IsNodeNonTerminal(node) == true, "")
}

func comparisonChecks(t *testing.T, chain *NodeChain, expectedOrder int, expectedCommonLabels int, expectedRelation g53.NameRelation) {
	if expectedOrder > 0 {
		ut.Assert(t, chain.lastComparison.Order > 0, "")
	} else if expectedOrder < 0 {
		ut.Assert(t, chain.lastComparison.Order < 0, "")
	} else {
		ut.Equal(t, chain.lastComparison.Order, 0)
	}

	ut.Equal(t, expectedCommonLabels, chain.lastComparison.CommonLabelCount)
	ut.Equal(t, expectedRelation, chain.lastComparison.Relation)
}

func TestTreeNodeChainLastComparison(t *testing.T) {
	chain := NewNodeChain()
	ut.Equal(t, chain.lastCompared, (*Node)(nil))

	emptyTree := NewDomainTree(false)
	node, ret := emptyTree.SearchExt(g53.NameFromStringUnsafe("a"), chain, nil, nil)
	ut.Equal(t, ret, NotFound)
	ut.Equal(t, chain.lastCompared, (*Node)(nil))
	chain.clear()

	tree := createDomainTree(true)
	node, _ = tree.SearchExt(g53.NameFromStringUnsafe("x.d.e.f"), chain, nil, nil)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, 0, 2, g53.EQUAL)
	chain.clear()

	_, ret = tree.Search(g53.NameFromStringUnsafe("i.g.h"))
	ut.Equal(t, ret, ExactMatch)
	node, ret = tree.SearchExt(g53.NameFromStringUnsafe("x.i.g.h"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, 1, 2, g53.SUBDOMAIN)
	chain.clear()

	// Partial match, search stopped in the subtree below the matching node
	// after following a left branch.
	node, ret = tree.Search(g53.NameFromStringUnsafe("x.d.e.f"))
	ut.Equal(t, ret, ExactMatch)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("a.d.e.f"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, -1, 1, g53.COMMONANCESTOR)
	chain.clear()

	// Partial match, search stopped in the subtree below the matching node
	// after following a right branch.
	node, ret = tree.Search(g53.NameFromStringUnsafe("z.d.e.f"))
	ut.Equal(t, ret, ExactMatch)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("zz.d.e.f"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, 1, 1, g53.COMMONANCESTOR)
	chain.clear()

	// Partial match, search stopped at a node for a super domain of the
	// search name in the subtree below the matching node.
	node, ret = tree.Search(g53.NameFromStringUnsafe("w.y.d.e.f"))
	ut.Equal(t, ret, ExactMatch)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("y.d.e.f"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, -1, 2, g53.SUPERDOMAIN)
	chain.clear()

	// Partial match, search stopped at a node that share a common ancestor
	// with the search name in the subtree below the matching node.
	// (the expected node is the same as the previous case)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("z.y.d.e.f"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, 1, 2, g53.COMMONANCESTOR)
	chain.clear()

	// Search stops in the highest level after following a left branch.
	node, ret = tree.Search(g53.NameFromStringUnsafe("c"))
	ut.Equal(t, ret, ExactMatch)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("bb"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	//ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, -1, 1, g53.COMMONANCESTOR)
	chain.clear()

	// Search stops in the highest level after following a right branch.
	// (the expected node is the same as the previous case)
	_, ret = tree.SearchExt(g53.NameFromStringUnsafe("d"), chain, nil, nil)
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, chain.lastCompared, node)
	comparisonChecks(t, chain, 1, 1, g53.COMMONANCESTOR)
	chain.clear()
}

func TestRootZone(t *testing.T) {
	tree := NewDomainTree(false)
	node, _ := treeInsertString(tree, "cn.")
	node.SetData(0)

	node, _ = treeInsertString(tree, ".")
	node.SetData(1)

	node, ret := tree.Search(g53.NameFromStringUnsafe("."))
	ut.Equal(t, ret, ExactMatch)

	node, ret = tree.Search(g53.NameFromStringUnsafe("example.com"))
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, node.name.String(true), ".")
	ut.Equal(t, node.Data().(int), 1)

	node, _ = treeInsertString(tree, "com")
	node.SetData(2)
	node, ret = tree.Search(g53.NameFromStringUnsafe("example.com"))
	ut.Equal(t, ret, PartialMatch)
	ut.Equal(t, node.name.String(true), "com")
	ut.Equal(t, node.Data().(int), 2)
}

func TestTreeForEach(t *testing.T) {
	tree := createDomainTree(true)
	nodeCount := 0
	tree.ForEach(func(n *Node) {
		nodeCount += 1
	})
	ut.Equal(t, tree.nodeCount, nodeCount)
}

func cloneIntPoint(v interface{}) interface{} {
	var n int
	n = *(v.(*int))
	return &n
}

func TestClone(t *testing.T) {
	tree := createDomainTree(true)
	new := tree.Clone(nil)
	ut.Equal(t, new.NodeCount(), tree.NodeCount())

	names := []string{
		"c", "b", "a", "x.d.e.f", "z.d.e.f", "g.h", "i.g.h", "o.w.y.d.e.f",
		"j.z.d.e.f", "p.w.y.d.e.f", "q.w.y.d.e.f"}

	for i, n := range names {
		node, _ := new.Search(g53.NameFromStringUnsafe(n))
		ut.Equal(t, i+1, node.Data().(int))
	}
}

func TestTreeRemove(t *testing.T) {
	tree := createDomainTree(true)
	names := []string{"c", "b", "a", "x.d.e.f", "z.d.e.f", "g.h", "i.g.h", "o.w.y.d.e.f", "j.z.d.e.f", "p.w.y.d.e.f", "q.w.y.d.e.f"}

	for _, n := range names {
		node, ret := tree.Search(g53.NameFromStringUnsafe(n))
		ut.Equal(t, ret, ExactMatch)
		err := tree.Remove(g53.NameFromStringUnsafe(n))
		ut.Equal(t, err, nil)
		_, ret = tree.Search(g53.NameFromStringUnsafe(n))
		ut.Equal(t, ret != ExactMatch, node.IsLeaf())
	}
	ut.Equal(t, 0, tree.NodeCount())
}

func TestTreeRemoveRandomDomain(t *testing.T) {
	tree := NewDomainTree(true)
	for i := 0; i < 100; i++ {
		err := removeRandDomain(tree)
		ut.Equal(t, err, nil)
	}
}

func removeRandDomain(tree *DomainTree) error {
	var names []*g53.Name
	for i := 1; i < 1000; i++ {
		name := g53.NameFromStringUnsafe(g53.RandomNoneFQDNDomain())
		node, err := tree.Insert(name)
		if err == nil {
			names = append(names, name)
			node.data = i
		}
	}

	for _, name := range names {
		node, ret := tree.Search(name)
		if ret != ExactMatch {
			return fmt.Errorf("no found name: %v in the tree\n", name.String(false))
		}
		err := tree.Remove(name)
		if err != nil {
			return fmt.Errorf("remove %s failed: %v\n", name.String(false), err.Error())
		}
		_, ret = tree.Search(name)
		if node.IsLeaf() && ret == ExactMatch {
			return fmt.Errorf("should remove name %s but found in tree\n", name.String(false))
		}
	}
	if 0 != tree.NodeCount() {
		return fmt.Errorf("should remove all name but found %v names in tree\n", tree.NodeCount())
	}
	return nil
}
