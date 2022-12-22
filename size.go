package main

type Size struct {
	Height uint64
	Width  uint64
}

func validWidth(width uint64) bool {
	if width == 100 || width == 200 || width == 400 || width == 600 || width == 800 || width == 1200 {
		return true
	}
	return false
}

func validHeight(height uint64) bool {
	if height == 100 || height == 200 || height == 400 || height == 600 || height == 800 || height == 1200 {
		return true
	}
	return false
}

func validSize(size Size) bool {
	if size.Height == 0 {
		return validWidth(size.Width)
	}

	if size.Width == 0 {
		return validHeight(size.Height)
	}

	return validWidth(size.Width) && validHeight(size.Height)
}
