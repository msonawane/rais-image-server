package magick

/*
#cgo pkg-config: MagickCore
#include <magick/MagickCore.h>
#include "magick.h"
*/
import "C"
import (
	"image"
	"reflect"
	"unsafe"
)

// Image returns a native Go image interface.  For now, this is always RGBA for
// simplicity, but it would be a good idea to use a gray image when it makes
// sense to improve performance and RAM usage.
func (i *Image) Image() (image.Image, error) {
	// Create and prep-for-freeing the exception
	exception := C.AcquireExceptionInfo()
	defer C.DestroyExceptionInfo(exception)

	img := image.NewRGBA(image.Rect(0, 0, i.decodeWidth, i.decodeHeight))

	area := i.decodeWidth * i.decodeHeight
	pixLen := area << 2
	pixels := make([]byte, pixLen)
	pi := reflect.ValueOf(pixels).Interface()
	ptr := unsafe.Pointer(&pixels[0])

	// Dimensions as C types
	w := C.size_t(i.decodeWidth)
	h := C.size_t(i.decodeHeight)

	C.ExportRGBA(i.image, w, h, ptr, exception)

	img.Pix = pi.([]uint8)

	return img, nil
}
