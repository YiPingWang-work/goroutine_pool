package monitor

var cnt uint64 = 0

func AddLoops() {
	cnt++
}

func ReturnLoops() uint64 {
	return cnt
}
