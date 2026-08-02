package ops

import (
	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/opsutil"
)

// dishText reads a dish as CyberChef reads one: as UTF-8 where that works,
// and otherwise one character per byte. Operations that walk their input as
// text must use this rather than the raw string, or a byte that is not valid
// UTF-8 becomes U+FFFD and its value is lost — 0xFF has to arrive as U+00FF
// for "To Charcode" to report 255 and for "To HTML Entity" to write &yuml;.
func dishText(in *core.Dish) string {
	return opsutil.BytesAsText(in.Bytes())
}
