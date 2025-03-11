package netipds

// HACK
//import (
//	"fmt"
//	"net/netip"
//	"testing"
//)
//
//// This is the README example copied and pasted; it's just a sanity check that
//// it compiles.
//func TestReadmeExampleLiteral(t *testing.T) {
//	/*** README snippet pasted below ***/
//
//	// Make our examples more readable
//	px := netip.MustParsePrefix
//
//	// Build a PrefixMap
//	builder := PrefixMapBuilder[string]{}
//	builder.Set(px("1.2.0.0/16"), "hello")
//	builder.Set(px("1.2.3.0/24"), "world")
//
//	// This returns an immutable snapshot of the
//	// builder's state. The builder remains usable.
//	pm := builder.PrefixMap()
//
//	// Fetch an exact entry from the PrefixMap.
//	val, ok := pm.Get(px("1.0.0.0/16")) // => ("hello", true)
//
//	// Ask if the PrefixMap contains an exact
//	// entry.
//	ok = pm.Contains(px("1.2.3.4/32")) // => false
//
//	// Ask if a Prefix has any ancestor in the
//	// PrefixMap.
//	ok = pm.Encompasses(px("1.2.3.4/32")) // => true
//
//	// Fetch a Prefix's nearest ancestor.
//	p, val, ok := pm.ParentOf(px("1.2.3.4/32")) // => (1.2.3.0/24, "world", true)
//
//	// Fetch all of a Prefix's ancestors, and
//	// convert the result to a map[Prefix]string.
//	m := pm.AncestorsOf(px("1.2.3.4/32")).ToMap() // => map[1.2.0.0/16:"hello"
//	//        1.2.3.0/24:"world"]
//
//	// Fetch all of a Prefix's descendants, and
//	// convert the result to a map[Prefix]string.
//	m = pm.DescendantsOf(px("1.0.0.0/8")).ToMap() // => map[1.2.0.0/16:"hello"
//	//        1.2.3.0/24:"world"]
//
//	/*** README snippet ends here ***/
//
//	// Appease the compiler
//	fmt.Println(val, ok, p, m)
//}
//
//// This is the code from the README example reimagined as an actual test case
//func TestReadmeExampleVerify(t *testing.T) {
//	// Make our examples more readable
//	px := netip.MustParsePrefix
//
//	// Build a PrefixMap
//	pmb := PrefixMapBuilder[string]{}
//	pmb.Set(px("1.2.0.0/16"), "hello")
//	pmb.Set(px("1.2.3.0/24"), "world")
//	pm := pmb.PrefixMap()
//
//	// Fetch an exact entry from the PrefixMap.
//	input := px("1.2.0.0/16")
//	valWant, okWant := "hello", true
//	val, ok := pm.Get(input)
//	if ok != true || val != "hello" {
//		t.Errorf("pm.Get(%s) = %v, %v, want %v, %v", input,
//			val, ok,
//			valWant, okWant,
//		)
//	}
//
//	// Ask if the PrefixMap contains an exact
//	// entry.
//	input = px("1.2.3.4/32")
//	ok = pm.Contains(input)
//	if ok != false {
//		t.Errorf("pm.Contains(%s) = %v, want %v", "1.2.3.4/32", ok, false)
//	}
//
//	// Ask if a Prefix has any ancestor in the
//	// PrefixMap.
//	ok = pm.Encompasses(px("1.2.3.4/32"))
//	if ok != true {
//		t.Errorf("pm.Encompasses(%s) = %v, want %v", "1.2.3.4/32", ok, true)
//	}
//
//	// Fetch a Prefix's nearest ancestor.
//	prefix := px("1.2.3.4/32")
//	pWant, valWant, okWant := px("1.2.3.0/24"), "world", true
//	p, val, ok := pm.ParentOf(prefix)
//	if p != pWant || val != valWant || ok != okWant {
//		t.Errorf("pm.ParentOf(%s) = %v, %v, %v, want %v, %v, %v",
//			prefix,
//			p, val, ok,
//			pWant, valWant, okWant,
//		)
//	}
//
//	want := map[netip.Prefix]string{
//		px("1.2.0.0/16"): "hello",
//		px("1.2.3.0/24"): "world",
//	}
//
//	// Fetch all of a Prefix's ancestors, and
//	// convert the result to a map[Prefix]string.
//	m := pm.AncestorsOf(px("1.2.3.4/32")).ToMap()
//	checkMap(t, want, m)
//
//	// Fetch all of a Prefix's descendants, and
//	// convert the result to a map[Prefix]string.
//	m = pm.DescendantsOf(px("1.0.0.0/8")).ToMap()
//	checkMap(t, want, m)
//}
