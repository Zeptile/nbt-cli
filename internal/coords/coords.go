package coords

import (
	"fmt"
)

func FloorDiv(a, b int) int {
	q := a / b
	r := a % b
	if (r != 0) && ((r > 0) != (b > 0)) {
		q--
	}
	return q
}

func FloorMod(a, b int) int {
	r := a % b
	if r < 0 {
		r += b
	}
	return r
}

func WorldToRegionXZ(x, z int) (int, int) {
	return FloorDiv(x, 512), FloorDiv(z, 512)
}

func RegionFileName(rx, rz int) string {
	return fmt.Sprintf("r.%d.%d.mca", rx, rz)
}

func WorldToChunkXZ(x, z int) (int, int) {
	return FloorDiv(x, 16), FloorDiv(z, 16)
}

func InRegionChunkIndex(cx, cz int) (int, int) {
	return FloorMod(cx, 32), FloorMod(cz, 32)
}

func LocalBlockXZ(x, z int) (int, int) {
	return FloorMod(x, 16), FloorMod(z, 16)
}


