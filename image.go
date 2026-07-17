package gltf

import (
	"bytes"
	"fmt"
	"image"
	"strings"

	// Register the PNG and JPEG decoders with image.Decode.
	_ "image/jpeg"
	_ "image/png"
)

// ImageBytes returns the raw encoded bytes of the image at imageIndex together
// with its MIME type. It resolves three sources:
//
//   - a base64 "data:" URI (MIME type taken from the URI);
//   - a bufferView-backed image (buffers must be resolved first);
//   - an external file, read relative to baseDir.
//
// baseDir may be empty when the image is embedded.
func (d *Document) ImageBytes(imageIndex int, baseDir string) ([]byte, string, error) {
	if imageIndex < 0 || imageIndex >= len(d.Images) {
		return nil, "", errIndexRange("image", imageIndex, len(d.Images))
	}
	img := &d.Images[imageIndex]
	switch {
	case img.BufferView != nil:
		_, buf, err := d.resolveBufferView(*img.BufferView)
		if err != nil {
			return nil, "", err
		}
		bv := &d.BufferViews[*img.BufferView]
		data := buf[bv.ByteOffset : bv.ByteOffset+bv.ByteLength]
		return data, img.MimeType, nil
	case strings.HasPrefix(img.URI, dataURIPrefix):
		data, err := decodeDataURI(img.URI)
		if err != nil {
			return nil, "", err
		}
		return data, dataURIMime(img.URI), nil
	case img.URI != "":
		data, err := readExternal(baseDir, img.URI)
		if err != nil {
			return nil, "", err
		}
		return data, img.MimeType, nil
	default:
		return nil, "", fmt.Errorf("gltf: image %d has neither uri nor bufferView", imageIndex)
	}
}

// DecodeImage decodes the image at imageIndex into an [image.Image], returning
// the registered format name (for example "png" or "jpeg"). PNG and JPEG are
// supported out of the box; register additional decoders via the standard
// library's image package to handle more formats. baseDir resolves external
// file URIs and may be empty for embedded images.
func (d *Document) DecodeImage(imageIndex int, baseDir string) (image.Image, string, error) {
	data, _, err := d.ImageBytes(imageIndex, baseDir)
	if err != nil {
		return nil, "", err
	}
	im, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("gltf: decoding image %d: %w", imageIndex, err)
	}
	return im, format, nil
}

// dataURIMime extracts the MIME type from a data URI, or returns an empty
// string when none is present.
func dataURIMime(uri string) string {
	comma := strings.IndexByte(uri, ',')
	if comma < 0 {
		return ""
	}
	meta := uri[len(dataURIPrefix):comma]
	if semi := strings.IndexByte(meta, ';'); semi >= 0 {
		meta = meta[:semi]
	}
	return meta
}
