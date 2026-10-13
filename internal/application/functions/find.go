package functions

import (
	"strings"

	"github.com/diwise/senml"
)

// FindByShortName matchar poster via kort eller kvalificerat namn. För
// råa pack är det identiskt med FindByName; för upplösta/berikade pack
// bär posterna fullständigt prefix (device/.../resurs). Hjälparen gör
// funktionsramverket tolerant mot labs-1.27:s berikade MessageAccepted
// utan att ändra beteendet för legacy-pack.
func FindByShortName(n string) senml.RecordFinder {
	return func(r senml.Record) bool {
		return r.Name == n || strings.HasSuffix(r.Name, "/"+n)
	}
}
