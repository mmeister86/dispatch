//go:build darwin && cgo

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit -framework Foundation
#import <AppKit/AppKit.h>
#import <Foundation/Foundation.h>
#include <stdlib.h>

static unsigned char* dispatch_clipboard_png(unsigned long long *length) {
	@autoreleasepool {
		NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
		NSData *pngData = [pasteboard dataForType:NSPasteboardTypePNG];
		if (pngData == nil) {
			NSData *tiffData = [pasteboard dataForType:NSPasteboardTypeTIFF];
			if (tiffData != nil) {
				NSBitmapImageRep *bitmap = [NSBitmapImageRep imageRepWithData:tiffData];
				if (bitmap != nil) {
					pngData = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
				}
			}
		}
		if (pngData == nil || [pngData length] == 0) {
			*length = 0;
			return NULL;
		}
		*length = [pngData length];
		unsigned char *buffer = (unsigned char*)malloc((size_t)*length);
		if (buffer == NULL) {
			*length = 0;
			return NULL;
		}
		memcpy(buffer, [pngData bytes], (size_t)*length);
		return buffer;
	}
}
*/
import "C"

import (
	"context"
	"unsafe"
)

func readNativeDarwinPNG(context.Context) ([]byte, error) {
	var length C.ulonglong
	ptr := C.dispatch_clipboard_png(&length)
	if ptr == nil || length == 0 {
		return nil, ErrNoImages
	}
	defer C.free(unsafe.Pointer(ptr))
	return C.GoBytes(unsafe.Pointer(ptr), C.int(length)), nil
}
