package filesig

// Code generated from CyberChef src/core/lib/FileSignatures.mjs. DO NOT EDIT.
// Regenerate via the generator in the scratchpad when the upstream table changes.

var Signatures = []Category{
	{Name: "Images", Types: []Sig{
		{
			Name: "Joint Photographic Experts Group image", Extension: "jpg,jpeg,jpe,thm,mpo",
			MIME: "image/jpeg", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xd8, hi: 0xd8}, {off: 2, lo: 0xff, hi: 0xff}, {off: 3, set: []byte{0xc0, 0xc4, 0xdb, 0xdd, 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe7, 0xe8, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xfe}}}},
			Carver: "JPEG",
		},
		{
			Name: "Graphics Interchange Format image", Extension: "gif",
			MIME: "image/gif", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x47, hi: 0x47}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x38, hi: 0x38}, {off: 4, set: []byte{0x37, 0x39}}, {off: 5, lo: 0x61, hi: 0x61}}},
			Carver: "GIF",
		},
		{
			Name: "Portable Network Graphics image", Extension: "png",
			MIME: "image/png", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x89, hi: 0x89}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x4e, hi: 0x4e}, {off: 3, lo: 0x47, hi: 0x47}, {off: 4, lo: 0xd, hi: 0xd}, {off: 5, lo: 0xa, hi: 0xa}, {off: 6, lo: 0x1a, hi: 0x1a}, {off: 7, lo: 0xa, hi: 0xa}}},
			Carver: "PNG",
		},
		{
			Name: "WEBP Image", Extension: "webp",
			MIME: "image/webp", Description: "",
			alts:   [][]sigCheck{{{off: 8, lo: 0x57, hi: 0x57}, {off: 9, lo: 0x45, hi: 0x45}, {off: 10, lo: 0x42, hi: 0x42}, {off: 11, lo: 0x50, hi: 0x50}}},
			Carver: "WEBP",
		},
		{
			Name: "High Efficiency Image File Format", Extension: "heic,heif",
			MIME: "image/heif", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, set: []byte{0x24, 0x18}}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x68, hi: 0x68}, {off: 9, lo: 0x65, hi: 0x65}, {off: 10, lo: 0x69, hi: 0x69}, {off: 11, lo: 0x63, hi: 0x63}}},
		},
		{
			Name: "Camera Image File Format", Extension: "crw",
			MIME: "image/x-canon-crw", Description: "",
			alts: [][]sigCheck{{{off: 6, lo: 0x48, hi: 0x48}, {off: 7, lo: 0x45, hi: 0x45}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x50, hi: 0x50}, {off: 10, lo: 0x43, hi: 0x43}, {off: 11, lo: 0x43, hi: 0x43}, {off: 12, lo: 0x44, hi: 0x44}, {off: 13, lo: 0x52, hi: 0x52}}},
		},
		{
			Name: "Canon CR2 raw image", Extension: "cr2",
			MIME: "image/x-canon-cr2", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x52, hi: 0x52}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x2a, hi: 0x2a}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x52, hi: 0x52}}},
		},
		{
			Name: "Tagged Image File Format image", Extension: "tif",
			MIME: "image/tiff", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x2a, hi: 0x2a}}},
		},
		{
			Name: "Bitmap image", Extension: "bmp",
			MIME: "image/bmp", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x42, hi: 0x42}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 7, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 14, set: []byte{0xc, 0x28, 0x38, 0x40, 0x6c, 0x7c}}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x0, hi: 0x0}, {off: 17, lo: 0x0, hi: 0x0}}},
			Carver: "BMP",
		},
		{
			Name: "JPEG Extended Range image", Extension: "jxr",
			MIME: "image/vnd.ms-photo", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0xbc, hi: 0xbc}}},
		},
		{
			Name: "Photoshop image", Extension: "psd",
			MIME: "image/vnd.adobe.photoshop", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x38, hi: 0x38}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Photoshop Large Document", Extension: "psb",
			MIME: "application/x-photoshop", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x38, hi: 0x38}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Paint Shop Pro image", Extension: "psp",
			MIME: "image/psp", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x69, hi: 0x69}, {off: 3, lo: 0x6e, hi: 0x6e}, {off: 4, lo: 0x74, hi: 0x74}, {off: 5, lo: 0x20, hi: 0x20}, {off: 6, lo: 0x53, hi: 0x53}, {off: 7, lo: 0x68, hi: 0x68}, {off: 8, lo: 0x6f, hi: 0x6f}, {off: 9, lo: 0x70, hi: 0x70}, {off: 10, lo: 0x20, hi: 0x20}, {off: 11, lo: 0x50, hi: 0x50}, {off: 12, lo: 0x72, hi: 0x72}, {off: 13, lo: 0x6f, hi: 0x6f}, {off: 14, lo: 0x20, hi: 0x20}, {off: 15, lo: 0x49, hi: 0x49}, {off: 16, lo: 0x6d, hi: 0x6d}}, {{off: 0, lo: 0x7e, hi: 0x7e}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x4b, hi: 0x4b}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "The GIMP image", Extension: "xcf",
			MIME: "image/x-xcf", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x67, hi: 0x67}, {off: 1, lo: 0x69, hi: 0x69}, {off: 2, lo: 0x6d, hi: 0x6d}, {off: 3, lo: 0x70, hi: 0x70}, {off: 4, lo: 0x20, hi: 0x20}, {off: 5, lo: 0x78, hi: 0x78}, {off: 6, lo: 0x63, hi: 0x63}, {off: 7, lo: 0x66, hi: 0x66}, {off: 8, lo: 0x20, hi: 0x20}, {off: 9, set: []byte{0x66, 0x76}}, {off: 10, set: []byte{0x69, 0x30}}, {off: 11, set: []byte{0x6c, 0x30}}, {off: 12, set: []byte{0x65, 0x31, 0x32, 0x33}}}},
		},
		{
			Name: "Icon image", Extension: "ico",
			MIME: "image/x-icon", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, set: []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15}}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, set: []byte{0x10, 0x20, 0x30, 0x40, 0x80}}, {off: 7, set: []byte{0x10, 0x20, 0x30, 0x40, 0x80}}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, set: []byte{0x0, 0x1}}}},
			Carver: "ICO",
		},
		{
			Name: "Radiance High Dynamic Range image", Extension: "hdr",
			MIME: "image/vnd.radiance", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x3f, hi: 0x3f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x41, hi: 0x41}, {off: 4, lo: 0x44, hi: 0x44}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x41, hi: 0x41}, {off: 7, lo: 0x4e, hi: 0x4e}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x45, hi: 0x45}, {off: 10, lo: 0xa, hi: 0xa}}},
		},
		{
			Name: "Sony ARW image", Extension: "arw",
			MIME: "image/x-raw", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x5, hi: 0x5}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x41, hi: 0x41}, {off: 5, lo: 0x57, hi: 0x57}, {off: 6, lo: 0x31, hi: 0x31}, {off: 7, lo: 0x2e, hi: 0x2e}}},
		},
		{
			Name: "Fujifilm Raw Image", Extension: "raf",
			MIME: "image/x-raw", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x55, hi: 0x55}, {off: 2, lo: 0x4a, hi: 0x4a}, {off: 3, lo: 0x49, hi: 0x49}, {off: 4, lo: 0x46, hi: 0x46}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x4c, hi: 0x4c}, {off: 7, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x43, hi: 0x43}, {off: 10, lo: 0x44, hi: 0x44}, {off: 11, lo: 0x2d, hi: 0x2d}, {off: 12, lo: 0x52, hi: 0x52}, {off: 13, lo: 0x41, hi: 0x41}, {off: 14, lo: 0x57, hi: 0x57}}},
		},
		{
			Name: "Minolta RAW image", Extension: "mrw",
			MIME: "image/x-raw", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}}},
		},
		{
			Name: "Adobe Bridge Thumbnail Cache", Extension: "bct",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x6c, hi: 0x6c}, {off: 1, lo: 0x6e, hi: 0x6e}, {off: 2, lo: 0x62, hi: 0x62}, {off: 3, lo: 0x74, hi: 0x74}, {off: 4, lo: 0x2, hi: 0x2}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Microsoft Document Imaging", Extension: "mdi",
			MIME: "image/vnd.ms-modi", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x45, hi: 0x45}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Joint Photographic Experts Group image (under Base64)", Extension: "B64",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x2f, hi: 0x2f}, {off: 1, lo: 0x39, hi: 0x39}, {off: 2, lo: 0x6a, hi: 0x6a}, {off: 3, lo: 0x2f, hi: 0x2f}, {off: 4, lo: 0x34, hi: 0x34}}},
		},
		{
			Name: "Portable Network Graphics image (under Base64)", Extension: "B64",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x69, hi: 0x69}, {off: 1, lo: 0x56, hi: 0x56}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x52, hi: 0x52}, {off: 5, lo: 0x77, hi: 0x77}, {off: 6, lo: 0x30, hi: 0x30}}},
		},
		{
			Name: "AutoCAD Drawing", Extension: "dwg,123d",
			MIME: "application/acad", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, set: []byte{0x30, 0x31}}, {off: 5, set: []byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35}}, {off: 6, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "AutoCAD Drawing", Extension: "dwg,dwt",
			MIME: "application/acad", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x31, hi: 0x31}, {off: 5, lo: 0x38, hi: 0x38}, {off: 6, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x32, hi: 0x32}, {off: 5, lo: 0x34, hi: 0x34}, {off: 6, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x32, hi: 0x32}, {off: 5, lo: 0x37, hi: 0x37}, {off: 6, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Targa Image", Extension: "tga",
			MIME: "image/x-targa", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x54, hi: 0x54}, {off: 1, lo: 0x52, hi: 0x52}, {off: 2, lo: 0x55, hi: 0x55}, {off: 3, lo: 0x45, hi: 0x45}, {off: 4, lo: 0x56, hi: 0x56}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x53, hi: 0x53}, {off: 7, lo: 0x49, hi: 0x49}, {off: 8, lo: 0x4f, hi: 0x4f}, {off: 9, lo: 0x4e, hi: 0x4e}, {off: 10, lo: 0x2d, hi: 0x2d}, {off: 11, lo: 0x58, hi: 0x58}, {off: 12, lo: 0x46, hi: 0x46}, {off: 13, lo: 0x49, hi: 0x49}, {off: 14, lo: 0x4c, hi: 0x4c}, {off: 15, lo: 0x45, hi: 0x45}, {off: 16, lo: 0x2e, hi: 0x2e}}},
			Carver: "TARGA",
		},
	}},
	{Name: "Video", Types: []Sig{
		{
			Name: "Matroska Multimedia Container", Extension: "mkv",
			MIME: "video/x-matroska", Description: "",
			alts: [][]sigCheck{{{off: 31, lo: 0x6d, hi: 0x6d}, {off: 32, lo: 0x61, hi: 0x61}, {off: 33, lo: 0x74, hi: 0x74}, {off: 34, lo: 0x72, hi: 0x72}, {off: 35, lo: 0x6f, hi: 0x6f}, {off: 36, lo: 0x73, hi: 0x73}, {off: 37, lo: 0x6b, hi: 0x6b}, {off: 38, lo: 0x61, hi: 0x61}}},
		},
		{
			Name: "WEBM video", Extension: "webm",
			MIME: "video/webm", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1a, hi: 0x1a}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0xdf, hi: 0xdf}, {off: 3, lo: 0xa3, hi: 0xa3}}},
		},
		{
			Name: "Flash MP4 video", Extension: "f4v",
			MIME: "video/mp4", Description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, set: []byte{0x66, 0x46}}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, set: []byte{0x76, 0x56}}, {off: 11, lo: 0x20, hi: 0x20}}},
		},
		{
			Name: "MPEG-4 video", Extension: "mp4",
			MIME: "video/mp4", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, set: []byte{0x18, 0x20}}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}}, {{off: 0, lo: 0x33, hi: 0x33}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x70, hi: 0x70}, {off: 3, lo: 0x35, hi: 0x35}}, {{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x1c, hi: 0x1c}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x6d, hi: 0x6d}, {off: 9, lo: 0x70, hi: 0x70}, {off: 10, lo: 0x34, hi: 0x34}, {off: 11, lo: 0x32, hi: 0x32}, {off: 16, lo: 0x6d, hi: 0x6d}, {off: 17, lo: 0x70, hi: 0x70}, {off: 18, lo: 0x34, hi: 0x34}, {off: 19, lo: 0x31, hi: 0x31}, {off: 20, lo: 0x6d, hi: 0x6d}, {off: 21, lo: 0x70, hi: 0x70}, {off: 22, lo: 0x34, hi: 0x34}, {off: 23, lo: 0x32, hi: 0x32}, {off: 24, lo: 0x69, hi: 0x69}, {off: 25, lo: 0x73, hi: 0x73}, {off: 26, lo: 0x6f, hi: 0x6f}, {off: 27, lo: 0x6d, hi: 0x6d}}},
		},
		{
			Name: "M4V video", Extension: "m4v",
			MIME: "video/x-m4v", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x1c, hi: 0x1c}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x4d, hi: 0x4d}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, lo: 0x56, hi: 0x56}}},
		},
		{
			Name: "Quicktime video", Extension: "mov",
			MIME: "video/quicktime", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x14, hi: 0x14}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}}},
		},
		{
			Name: "Audio Video Interleave", Extension: "avi",
			MIME: "video/x-msvideo", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x56, hi: 0x56}, {off: 10, lo: 0x49, hi: 0x49}}},
		},
		{
			Name: "Windows Media Video", Extension: "wmv",
			MIME: "video/x-ms-wmv", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x26, hi: 0x26}, {off: 2, lo: 0xb2, hi: 0xb2}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x8e, hi: 0x8e}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0xcf, hi: 0xcf}, {off: 7, lo: 0x11, hi: 0x11}, {off: 8, lo: 0xa6, hi: 0xa6}, {off: 9, lo: 0xd9, hi: 0xd9}}},
		},
		{
			Name: "MPEG video", Extension: "mpg",
			MIME: "video/mpeg", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0xba, hi: 0xba}}},
		},
		{
			Name: "Flash Video", Extension: "flv",
			MIME: "video/x-flv", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x56, hi: 0x56}, {off: 3, lo: 0x1, hi: 0x1}}},
			Carver: "FLV",
		},
		{
			Name: "OGG Video", Extension: "ogv,ogm,opus,ogx",
			MIME: "video/ogg", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x1, hi: 0x1}, {off: 29, lo: 0x76, hi: 0x76}, {off: 30, lo: 0x69, hi: 0x69}, {off: 31, lo: 0x64, hi: 0x64}, {off: 32, lo: 0x65, hi: 0x65}, {off: 33, lo: 0x6f, hi: 0x6f}}, {{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x80, hi: 0x80}, {off: 29, lo: 0x74, hi: 0x74}, {off: 30, lo: 0x68, hi: 0x68}, {off: 31, lo: 0x65, hi: 0x65}, {off: 32, lo: 0x6f, hi: 0x6f}, {off: 33, lo: 0x72, hi: 0x72}, {off: 34, lo: 0x61, hi: 0x61}}, {{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x66, hi: 0x66}, {off: 29, lo: 0x69, hi: 0x69}, {off: 30, lo: 0x73, hi: 0x73}, {off: 31, lo: 0x68, hi: 0x68}, {off: 32, lo: 0x65, hi: 0x65}, {off: 33, lo: 0x61, hi: 0x61}, {off: 34, lo: 0x64, hi: 0x64}}},
		},
	}},
	{Name: "Audio", Types: []Sig{
		{
			Name: "Waveform Audio", Extension: "wav",
			MIME: "audio/x-wav", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 8, lo: 0x57, hi: 0x57}, {off: 9, lo: 0x41, hi: 0x41}, {off: 10, lo: 0x56, hi: 0x56}, {off: 11, lo: 0x45, hi: 0x45}}},
			Carver: "WAV",
		},
		{
			Name: "OGG audio", Extension: "ogg",
			MIME: "audio/ogg", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}}},
		},
		{
			Name: "Musical Instrument Digital Interface audio", Extension: "midi",
			MIME: "audio/midi", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x68, hi: 0x68}, {off: 3, lo: 0x64, hi: 0x64}}},
		},
		{
			Name: "MPEG-3 audio", Extension: "mp3",
			MIME: "audio/mpeg", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x44, hi: 0x44}, {off: 2, lo: 0x33, hi: 0x33}}, {{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfb, hi: 0xfb}}},
			Carver: "MP3",
		},
		{
			Name: "MPEG-4 Part 14 audio", Extension: "m4a",
			MIME: "audio/m4a", Description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x4d, hi: 0x4d}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, lo: 0x41, hi: 0x41}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x34, hi: 0x34}, {off: 2, lo: 0x41, hi: 0x41}, {off: 3, lo: 0x20, hi: 0x20}}},
		},
		{
			Name: "Free Lossless Audio Codec", Extension: "flac",
			MIME: "audio/x-flac", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x66, hi: 0x66}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x61, hi: 0x61}, {off: 3, lo: 0x43, hi: 0x43}}},
		},
		{
			Name: "Adaptive Multi-Rate audio codec", Extension: "amr",
			MIME: "audio/amr", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x41, hi: 0x41}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 4, lo: 0x52, hi: 0x52}, {off: 5, lo: 0xa, hi: 0xa}}},
		},
		{
			Name: "Audacity", Extension: "au",
			MIME: "audio/x-au", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x6e, hi: 0x6e}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x2e, hi: 0x2e}, {off: 24, lo: 0x41, hi: 0x41}, {off: 25, lo: 0x75, hi: 0x75}, {off: 26, lo: 0x64, hi: 0x64}, {off: 27, lo: 0x61, hi: 0x61}, {off: 28, lo: 0x63, hi: 0x63}, {off: 29, lo: 0x69, hi: 0x69}, {off: 30, lo: 0x74, hi: 0x74}, {off: 31, lo: 0x79, hi: 0x79}, {off: 32, lo: 0x42, hi: 0x42}, {off: 33, lo: 0x6c, hi: 0x6c}, {off: 34, lo: 0x6f, hi: 0x6f}, {off: 35, lo: 0x63, hi: 0x63}, {off: 36, lo: 0x6b, hi: 0x6b}, {off: 37, lo: 0x46, hi: 0x46}, {off: 38, lo: 0x69, hi: 0x69}, {off: 39, lo: 0x6c, hi: 0x6c}, {off: 40, lo: 0x65, hi: 0x65}}},
		},
		{
			Name: "Audacity Block", Extension: "auf",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x75, hi: 0x75}, {off: 2, lo: 0x64, hi: 0x64}, {off: 3, lo: 0x61, hi: 0x61}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x69, hi: 0x69}, {off: 6, lo: 0x74, hi: 0x74}, {off: 7, lo: 0x79, hi: 0x79}, {off: 8, lo: 0x42, hi: 0x42}, {off: 9, lo: 0x6c, hi: 0x6c}, {off: 10, lo: 0x6f, hi: 0x6f}, {off: 11, lo: 0x63, hi: 0x63}, {off: 12, lo: 0x6b, hi: 0x6b}, {off: 13, lo: 0x46, hi: 0x46}, {off: 14, lo: 0x69, hi: 0x69}, {off: 15, lo: 0x6c, hi: 0x6c}, {off: 16, lo: 0x65, hi: 0x65}}},
		},
		{
			Name: "Audio Interchange File", Extension: "aif",
			MIME: "audio/x-aiff", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x49, hi: 0x49}, {off: 10, lo: 0x46, hi: 0x46}, {off: 11, lo: 0x46, hi: 0x46}}},
		},
		{
			Name: "Audio Interchange File (compressed)", Extension: "aifc",
			MIME: "audio/x-aifc", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x49, hi: 0x49}, {off: 10, lo: 0x46, hi: 0x46}, {off: 11, lo: 0x43, hi: 0x43}}},
		},
	}},
	{Name: "Documents", Types: []Sig{
		{
			Name: "Portable Document Format", Extension: "pdf",
			MIME: "application/pdf", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x44, hi: 0x44}, {off: 3, lo: 0x46, hi: 0x46}}},
			Carver: "PDF",
		},
		{
			Name: "Portable Document Format (under Base64)", Extension: "B64",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x4a, hi: 0x4a}, {off: 2, lo: 0x56, hi: 0x56}, {off: 3, lo: 0x42, hi: 0x42}, {off: 4, lo: 0x45, hi: 0x45}, {off: 5, lo: 0x52, hi: 0x52}, {off: 6, lo: 0x69, hi: 0x69}}},
		},
		{
			Name: "Adobe PostScript", Extension: "ps,eps,ai,pfa",
			MIME: "application/postscript", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x2d, hi: 0x2d}, {off: 5, lo: 0x41, hi: 0x41}, {off: 6, lo: 0x64, hi: 0x64}, {off: 7, lo: 0x6f, hi: 0x6f}, {off: 8, lo: 0x62, hi: 0x62}, {off: 9, lo: 0x65, hi: 0x65}}},
		},
		{
			Name: "PostScript", Extension: "ps",
			MIME: "application/postscript", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x21, hi: 0x21}}},
		},
		{
			Name: "Encapsulated PostScript", Extension: "eps,ai",
			MIME: "application/eps", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xc5, hi: 0xc5}, {off: 1, lo: 0xd0, hi: 0xd0}, {off: 2, lo: 0xd3, hi: 0xd3}, {off: 3, lo: 0xc6, hi: 0xc6}}},
		},
		{
			Name: "Rich Text Format", Extension: "rtf",
			MIME: "application/rtf", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x7b, hi: 0x7b}, {off: 1, lo: 0x5c, hi: 0x5c}, {off: 2, lo: 0x72, hi: 0x72}, {off: 3, lo: 0x74, hi: 0x74}}},
			Carver: "RTF",
		},
		{
			Name: "Microsoft Office document/OLE2", Extension: "ole2,doc,xls,dot,ppt,xla,ppa,pps,pot,msi,sdw,db,vsd,msg",
			MIME: "application/msword,application/vnd.ms-excel,application/vnd.ms-powerpoint", Description: "Microsoft Office documents",
			alts: [][]sigCheck{{{off: 0, lo: 0xd0, hi: 0xd0}, {off: 1, lo: 0xcf, hi: 0xcf}, {off: 2, lo: 0x11, hi: 0x11}, {off: 3, lo: 0xe0, hi: 0xe0}, {off: 4, lo: 0xa1, hi: 0xa1}, {off: 5, lo: 0xb1, hi: 0xb1}, {off: 6, lo: 0x1a, hi: 0x1a}, {off: 7, lo: 0xe1, hi: 0xe1}}},
		},
		{
			Name: "Microsoft Office document/OLE2 (under Base64)", Extension: "B64",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x38, hi: 0x38}, {off: 3, lo: 0x52, hi: 0x52}, {off: 4, lo: 0x34, hi: 0x34}, {off: 5, lo: 0x4b, hi: 0x4b}, {off: 6, lo: 0x47, hi: 0x47}, {off: 7, lo: 0x78, hi: 0x78}}},
		},
		{
			Name: "Microsoft Office 2007+ document", Extension: "docx,xlsx,pptx",
			MIME: "application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.openxmlformats-officedocument.presentationml.presentation", Description: "",
			alts:   [][]sigCheck{{{off: 38, lo: 0x5f, hi: 0x5f}, {off: 39, lo: 0x54, hi: 0x54}, {off: 40, lo: 0x79, hi: 0x79}, {off: 41, lo: 0x70, hi: 0x70}, {off: 42, lo: 0x65, hi: 0x65}, {off: 43, lo: 0x73, hi: 0x73}, {off: 44, lo: 0x5d, hi: 0x5d}, {off: 45, lo: 0x2e, hi: 0x2e}, {off: 46, lo: 0x78, hi: 0x78}, {off: 47, lo: 0x6d, hi: 0x6d}, {off: 48, lo: 0x6c, hi: 0x6c}}},
			Carver: "ZIP",
		},
		{
			Name: "Microsoft Access database", Extension: "mdb,mda,mde,mdt,fdb,psa",
			MIME: "application/msaccess", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x61, hi: 0x61}, {off: 7, lo: 0x6e, hi: 0x6e}, {off: 8, lo: 0x64, hi: 0x64}, {off: 9, lo: 0x61, hi: 0x61}, {off: 10, lo: 0x72, hi: 0x72}, {off: 11, lo: 0x64, hi: 0x64}, {off: 12, lo: 0x20, hi: 0x20}, {off: 13, lo: 0x4a, hi: 0x4a}, {off: 14, lo: 0x65, hi: 0x65}, {off: 15, lo: 0x74, hi: 0x74}}},
		},
		{
			Name: "Microsoft Access 2007+ database", Extension: "accdb,accde,accda,accdu",
			MIME: "application/msaccess", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x61, hi: 0x61}, {off: 7, lo: 0x6e, hi: 0x6e}, {off: 8, lo: 0x64, hi: 0x64}, {off: 9, lo: 0x61, hi: 0x61}, {off: 10, lo: 0x72, hi: 0x72}, {off: 11, lo: 0x64, hi: 0x64}, {off: 12, lo: 0x20, hi: 0x20}, {off: 13, lo: 0x41, hi: 0x41}, {off: 14, lo: 0x43, hi: 0x43}, {off: 15, lo: 0x45, hi: 0x45}, {off: 16, lo: 0x20, hi: 0x20}}},
		},
		{
			Name: "Microsoft OneNote document", Extension: "one",
			MIME: "application/onenote", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xe4, hi: 0xe4}, {off: 1, lo: 0x52, hi: 0x52}, {off: 2, lo: 0x5c, hi: 0x5c}, {off: 3, lo: 0x7b, hi: 0x7b}, {off: 4, lo: 0x8c, hi: 0x8c}, {off: 5, lo: 0xd8, hi: 0xd8}, {off: 6, lo: 0xa7, hi: 0xa7}, {off: 7, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0xae, hi: 0xae}, {off: 9, lo: 0xb1, hi: 0xb1}, {off: 10, lo: 0x53, hi: 0x53}, {off: 11, lo: 0x78, hi: 0x78}, {off: 12, lo: 0xd0, hi: 0xd0}, {off: 13, lo: 0x29, hi: 0x29}, {off: 14, lo: 0x96, hi: 0x96}, {off: 15, lo: 0xd3, hi: 0xd3}}},
		},
		{
			Name: "Outlook Express database", Extension: "dbx",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xcf, hi: 0xcf}, {off: 1, lo: 0xad, hi: 0xad}, {off: 2, lo: 0x12, hi: 0x12}, {off: 3, lo: 0xfe, hi: 0xfe}, {off: 4, set: []byte{0x30, 0xc5, 0xc6, 0xc7}}, {off: 11, lo: 0x11, hi: 0x11}}},
		},
		{
			Name: "Personal Storage Table (Outlook)", Extension: "pst,ost,fdb,pab",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x21, hi: 0x21}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x44, hi: 0x44}, {off: 3, lo: 0x4e, hi: 0x4e}}},
		},
		{
			Name: "Microsoft Exchange Database", Extension: "edb",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0xef, hi: 0xef}, {off: 5, lo: 0xcd, hi: 0xcd}, {off: 6, lo: 0xab, hi: 0xab}, {off: 7, lo: 0x89, hi: 0x89}, {off: 8, set: []byte{0x20, 0x23}}, {off: 9, lo: 0x6, hi: 0x6}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, set: []byte{0x0, 0x1}}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x0, hi: 0x0}, {off: 15, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "WordPerfect document", Extension: "wpd,wp,wp5,wp6,wpp,bk!,wcm",
			MIME: "application/wordperfect", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0x57, hi: 0x57}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x43, hi: 0x43}, {off: 7, set: []byte{0x0, 0x1, 0x2}}, {off: 8, lo: 0x1, hi: 0x1}, {off: 9, lo: 0xa, hi: 0xa}}},
		},
		{
			Name: "EPUB e-book", Extension: "epub",
			MIME: "application/epub+zip", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x3, hi: 0x3}, {off: 3, lo: 0x4, hi: 0x4}, {off: 30, lo: 0x6d, hi: 0x6d}, {off: 31, lo: 0x69, hi: 0x69}, {off: 32, lo: 0x6d, hi: 0x6d}, {off: 33, lo: 0x65, hi: 0x65}, {off: 34, lo: 0x74, hi: 0x74}, {off: 35, lo: 0x79, hi: 0x79}, {off: 36, lo: 0x70, hi: 0x70}, {off: 37, lo: 0x65, hi: 0x65}, {off: 38, lo: 0x61, hi: 0x61}, {off: 39, lo: 0x70, hi: 0x70}, {off: 40, lo: 0x70, hi: 0x70}, {off: 41, lo: 0x6c, hi: 0x6c}, {off: 42, lo: 0x69, hi: 0x69}, {off: 43, lo: 0x63, hi: 0x63}, {off: 44, lo: 0x61, hi: 0x61}, {off: 45, lo: 0x74, hi: 0x74}, {off: 46, lo: 0x69, hi: 0x69}, {off: 47, lo: 0x6f, hi: 0x6f}, {off: 48, lo: 0x6e, hi: 0x6e}, {off: 49, lo: 0x2f, hi: 0x2f}, {off: 50, lo: 0x65, hi: 0x65}, {off: 51, lo: 0x70, hi: 0x70}, {off: 52, lo: 0x75, hi: 0x75}, {off: 53, lo: 0x62, hi: 0x62}, {off: 54, lo: 0x2b, hi: 0x2b}, {off: 55, lo: 0x7a, hi: 0x7a}, {off: 56, lo: 0x69, hi: 0x69}, {off: 57, lo: 0x70, hi: 0x70}}},
			Carver: "ZIP",
		},
	}},
	{Name: "Applications", Types: []Sig{
		{
			Name: "Windows Portable Executable", Extension: "exe,dll,drv,vxd,sys,ocx,vbx,com,fon,scr",
			MIME: "application/vnd.microsoft.portable-executable", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x5a, hi: 0x5a}, {off: 3, set: []byte{0x0, 0x1, 0x2}}, {off: 5, set: []byte{0x0, 0x1, 0x2}}}},
			Carver: "MZPE",
		},
		{
			Name: "Executable and Linkable Format", Extension: "elf,bin,axf,o,prx,so",
			MIME: "application/x-executable", Description: "Executable and Linkable Format file. No standard file extension.",
			alts:   [][]sigCheck{{{off: 0, lo: 0x7f, hi: 0x7f}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0x4c, hi: 0x4c}, {off: 3, lo: 0x46, hi: 0x46}}},
			Carver: "ELF",
		},
		{
			Name: "MacOS Mach-O object", Extension: "dylib",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xca, hi: 0xca}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0xba, hi: 0xba}, {off: 3, lo: 0xbe, hi: 0xbe}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, set: []byte{0x1, 0x2, 0x3}}}, {{off: 0, lo: 0xce, hi: 0xce}, {off: 1, lo: 0xfa, hi: 0xfa}, {off: 2, lo: 0xed, hi: 0xed}, {off: 3, lo: 0xfe, hi: 0xfe}, {off: 4, lo: 0x7, hi: 0x7}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, set: []byte{0x1, 0x2, 0x3}}}},
			Carver: "MACHO",
		},
		{
			Name: "MacOS Mach-O 64-bit object", Extension: "dylib",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xcf, hi: 0xcf}, {off: 1, lo: 0xfa, hi: 0xfa}, {off: 2, lo: 0xed, hi: 0xed}, {off: 3, lo: 0xfe, hi: 0xfe}}},
			Carver: "MACHO",
		},
		{
			Name: "Adobe Flash", Extension: "swf",
			MIME: "application/x-shockwave-flash", Description: "",
			alts: [][]sigCheck{{{off: 0, set: []byte{0x43, 0x46}}, {off: 1, lo: 0x57, hi: 0x57}, {off: 2, lo: 0x53, hi: 0x53}}},
		},
		{
			Name: "Java Class", Extension: "class",
			MIME: "application/java-vm", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xca, hi: 0xca}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0xba, hi: 0xba}, {off: 3, lo: 0xbe, hi: 0xbe}}},
		},
		{
			Name: "Dalvik Executable", Extension: "dex",
			MIME: "application/octet-stream", Description: "Dalvik Executable as used by Android",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x65, hi: 0x65}, {off: 2, lo: 0x78, hi: 0x78}, {off: 3, lo: 0xa, hi: 0xa}, {off: 4, lo: 0x30, hi: 0x30}, {off: 5, lo: 0x33, hi: 0x33}, {off: 6, lo: 0x35, hi: 0x35}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Google Chrome Extension", Extension: "crx",
			MIME: "application/crx", Description: "Google Chrome extension or packaged app",
			alts: [][]sigCheck{{{off: 0, lo: 0x43, hi: 0x43}, {off: 1, lo: 0x72, hi: 0x72}, {off: 2, lo: 0x32, hi: 0x32}, {off: 3, lo: 0x34, hi: 0x34}}},
		},
	}},
	{Name: "Archives", Types: []Sig{
		{
			Name: "PKZIP archive", Extension: "zip",
			MIME: "application/zip", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, set: []byte{0x3, 0x5, 0x7}}, {off: 3, set: []byte{0x4, 0x6, 0x8}}}},
			Carver: "ZIP",
		},
		{
			Name: "PKZIP archive (under Base64)", Extension: "B64",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x55, hi: 0x55}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x44, hi: 0x44}, {off: 4, lo: 0x42, hi: 0x42}, {off: 5, lo: 0x42, hi: 0x42}}},
		},
		{
			Name: "TAR archive", Extension: "tar",
			MIME: "application/x-tar", Description: "",
			alts:   [][]sigCheck{{{off: 257, lo: 0x75, hi: 0x75}, {off: 258, lo: 0x73, hi: 0x73}, {off: 259, lo: 0x74, hi: 0x74}, {off: 260, lo: 0x61, hi: 0x61}, {off: 261, lo: 0x72, hi: 0x72}}},
			Carver: "TAR",
		},
		{
			Name: "Roshal Archive", Extension: "rar",
			MIME: "application/x-rar-compressed", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x72, hi: 0x72}, {off: 3, lo: 0x21, hi: 0x21}, {off: 4, lo: 0x1a, hi: 0x1a}, {off: 5, lo: 0x7, hi: 0x7}, {off: 6, set: []byte{0x0, 0x1}}}},
		},
		{
			Name: "Gzip", Extension: "gz",
			MIME: "application/gzip", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x1f, hi: 0x1f}, {off: 1, lo: 0x8b, hi: 0x8b}, {off: 2, lo: 0x8, hi: 0x8}}},
			Carver: "GZIP",
		},
		{
			Name: "Bzip2", Extension: "bz2",
			MIME: "application/x-bzip2", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x42, hi: 0x42}, {off: 1, lo: 0x5a, hi: 0x5a}, {off: 2, lo: 0x68, hi: 0x68}}},
			Carver: "BZIP2",
		},
		{
			Name: "7zip", Extension: "7z",
			MIME: "application/x-7z-compressed", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x37, hi: 0x37}, {off: 1, lo: 0x7a, hi: 0x7a}, {off: 2, lo: 0xbc, hi: 0xbc}, {off: 3, lo: 0xaf, hi: 0xaf}, {off: 4, lo: 0x27, hi: 0x27}, {off: 5, lo: 0x1c, hi: 0x1c}}},
		},
		{
			Name: "Zlib Deflate", Extension: "zlib",
			MIME: "application/x-deflate", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, set: []byte{0x1, 0x9c, 0xda, 0x5e}}}},
			Carver: "Zlib",
		},
		{
			Name: "xz compression", Extension: "xz",
			MIME: "application/x-xz", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xfd, hi: 0xfd}, {off: 1, lo: 0x37, hi: 0x37}, {off: 2, lo: 0x7a, hi: 0x7a}, {off: 3, lo: 0x58, hi: 0x58}, {off: 4, lo: 0x5a, hi: 0x5a}, {off: 5, lo: 0x0, hi: 0x0}}},
			Carver: "XZ",
		},
		{
			Name: "Tarball", Extension: "tar.z",
			MIME: "application/x-gtar", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1f, hi: 0x1f}, {off: 1, set: []byte{0x9d, 0xa0}}}},
		},
		{
			Name: "ISO disk image", Extension: "iso",
			MIME: "application/octet-stream", Description: "ISO 9660 CD/DVD image file",
			alts: [][]sigCheck{{{off: 32769, lo: 0x43, hi: 0x43}, {off: 32770, lo: 0x44, hi: 0x44}, {off: 32771, lo: 0x30, hi: 0x30}, {off: 32772, lo: 0x30, hi: 0x30}, {off: 32773, lo: 0x31, hi: 0x31}}, {{off: 34817, lo: 0x43, hi: 0x43}, {off: 34818, lo: 0x44, hi: 0x44}, {off: 34819, lo: 0x30, hi: 0x30}, {off: 34820, lo: 0x30, hi: 0x30}, {off: 34821, lo: 0x31, hi: 0x31}}, {{off: 36865, lo: 0x43, hi: 0x43}, {off: 36866, lo: 0x44, hi: 0x44}, {off: 36867, lo: 0x30, hi: 0x30}, {off: 36868, lo: 0x30, hi: 0x30}, {off: 36869, lo: 0x31, hi: 0x31}}},
		},
		{
			Name: "Virtual Machine Disk", Extension: "vmdk",
			MIME: "application/vmdk,application/x-virtualbox-vmdk", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4b, hi: 0x4b}, {off: 1, lo: 0x44, hi: 0x44}, {off: 2, lo: 0x4d, hi: 0x4d}, {off: 3, lo: 0x56, hi: 0x56}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Virtual Hard Drive", Extension: "vhd",
			MIME: "application/x-vhd", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x63, hi: 0x63}, {off: 1, lo: 0x6f, hi: 0x6f}, {off: 2, lo: 0x6e, hi: 0x6e}, {off: 3, lo: 0x65, hi: 0x65}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x69, hi: 0x69}, {off: 7, lo: 0x78, hi: 0x78}}},
		},
		{
			Name: "Macintosh disk image", Extension: "dmf,dmg",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0xd, hi: 0xd}, {off: 4, lo: 0x62, hi: 0x62}, {off: 5, lo: 0x62, hi: 0x62}, {off: 6, lo: 0x60, hi: 0x60}, {off: 7, lo: 0x60, hi: 0x60}}},
		},
		{
			Name: "ARJ Archive", Extension: "arj",
			MIME: "application/x-arj-compressed", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x60, hi: 0x60}, {off: 1, lo: 0xea, hi: 0xea}, {off: 8, set: []byte{0x0, 0x10, 0x14}}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x2, hi: 0x2}}},
		},
		{
			Name: "WinAce Archive", Extension: "ace",
			MIME: "application/x-ace-compressed", Description: "",
			alts: [][]sigCheck{{{off: 7, lo: 0x2a, hi: 0x2a}, {off: 8, lo: 0x2a, hi: 0x2a}, {off: 9, lo: 0x41, hi: 0x41}, {off: 10, lo: 0x43, hi: 0x43}, {off: 11, lo: 0x45, hi: 0x45}, {off: 12, lo: 0x2a, hi: 0x2a}, {off: 13, lo: 0x2a, hi: 0x2a}}},
		},
		{
			Name: "Macintosh BinHex Encoded File", Extension: "hqx",
			MIME: "application/mac-binhex", Description: "",
			alts: [][]sigCheck{{{off: 11, lo: 0x6d, hi: 0x6d}, {off: 12, lo: 0x75, hi: 0x75}, {off: 13, lo: 0x73, hi: 0x73}, {off: 14, lo: 0x74, hi: 0x74}, {off: 15, lo: 0x20, hi: 0x20}, {off: 16, lo: 0x62, hi: 0x62}, {off: 17, lo: 0x65, hi: 0x65}, {off: 18, lo: 0x20, hi: 0x20}, {off: 19, lo: 0x63, hi: 0x63}, {off: 20, lo: 0x6f, hi: 0x6f}, {off: 21, lo: 0x6e, hi: 0x6e}, {off: 22, lo: 0x76, hi: 0x76}, {off: 23, lo: 0x65, hi: 0x65}, {off: 24, lo: 0x72, hi: 0x72}, {off: 25, lo: 0x74, hi: 0x74}, {off: 26, lo: 0x65, hi: 0x65}, {off: 27, lo: 0x64, hi: 0x64}, {off: 28, lo: 0x20, hi: 0x20}, {off: 29, lo: 0x77, hi: 0x77}, {off: 30, lo: 0x69, hi: 0x69}, {off: 31, lo: 0x74, hi: 0x74}, {off: 32, lo: 0x68, hi: 0x68}, {off: 33, lo: 0x20, hi: 0x20}, {off: 34, lo: 0x42, hi: 0x42}, {off: 35, lo: 0x69, hi: 0x69}, {off: 36, lo: 0x6e, hi: 0x6e}, {off: 37, lo: 0x48, hi: 0x48}, {off: 38, lo: 0x65, hi: 0x65}, {off: 39, lo: 0x78, hi: 0x78}}},
		},
		{
			Name: "ALZip Archive", Extension: "alz",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x5a, hi: 0x5a}, {off: 3, lo: 0x1, hi: 0x1}, {off: 4, lo: 0xa, hi: 0xa}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "KGB Compressed Archive", Extension: "kgb",
			MIME: "application/x-kgb-compressed", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4b, hi: 0x4b}, {off: 1, lo: 0x47, hi: 0x47}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x5f, hi: 0x5f}, {off: 4, lo: 0x61, hi: 0x61}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x63, hi: 0x63}, {off: 7, lo: 0x68, hi: 0x68}, {off: 8, lo: 0x20, hi: 0x20}, {off: 9, lo: 0x2d, hi: 0x2d}}},
		},
		{
			Name: "Microsoft Cabinet", Extension: "cab",
			MIME: "vnd.ms-cab-compressed", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x53, hi: 0x53}, {off: 2, lo: 0x43, hi: 0x43}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Jar Archive", Extension: "jar",
			MIME: "application/java-archive", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x5f, hi: 0x5f}, {off: 1, lo: 0x27, hi: 0x27}, {off: 2, lo: 0xa8, hi: 0xa8}, {off: 3, lo: 0x89, hi: 0x89}}},
		},
		{
			Name: "Jar Archive", Extension: "jar",
			MIME: "application/java-archive", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x3, hi: 0x3}, {off: 3, lo: 0x4, hi: 0x4}, {off: 4, lo: 0x14, hi: 0x14}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x8, hi: 0x8}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x8, hi: 0x8}, {off: 9, lo: 0x0, hi: 0x0}}},
			Carver: "ZIP",
		},
		{
			Name: "lzop compressed", Extension: "lzop,lzo",
			MIME: "application/x-lzop", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x89, hi: 0x89}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x5a, hi: 0x5a}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0xd, hi: 0xd}, {off: 6, lo: 0xa, hi: 0xa}, {off: 7, lo: 0x1a, hi: 0x1a}}},
			Carver: "LZOP",
		},
		{
			Name: "Linux deb package", Extension: "deb",
			MIME: "application/vnd.debian.binary-package", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x21, hi: 0x21}, {off: 1, lo: 0x3c, hi: 0x3c}, {off: 2, lo: 0x61, hi: 0x61}, {off: 3, lo: 0x72, hi: 0x72}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x68, hi: 0x68}, {off: 6, lo: 0x3e, hi: 0x3e}}},
			Carver: "DEB",
		},
		{
			Name: "Apple Disk Image", Extension: "dmg",
			MIME: "application/x-apple-diskimage", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0xd, hi: 0xd}, {off: 4, lo: 0x62, hi: 0x62}, {off: 5, lo: 0x62, hi: 0x62}, {off: 6, lo: 0x60, hi: 0x60}}},
		},
	}},
	{Name: "Miscellaneous", Types: []Sig{
		{
			Name: "UTF-8 text", Extension: "txt",
			MIME: "text/plain", Description: "UTF-8 encoded Unicode byte order mark, commonly but not exclusively seen in text files.",
			alts: [][]sigCheck{{{off: 0, lo: 0xef, hi: 0xef}, {off: 1, lo: 0xbb, hi: 0xbb}, {off: 2, lo: 0xbf, hi: 0xbf}}},
		},
		{
			Name: "UTF-32 LE text", Extension: "utf32le",
			MIME: "charset/utf32le", Description: "Little-endian UTF-32 encoded Unicode byte order mark.",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "UTF-16 LE text", Extension: "utf16le",
			MIME: "charset/utf16le", Description: "Little-endian UTF-16 encoded Unicode byte order mark.",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}}},
		},
		{
			Name: "Web Open Font Format", Extension: "woff",
			MIME: "application/font-woff", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x77, hi: 0x77}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Web Open Font Format 2", Extension: "woff2",
			MIME: "application/font-woff", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x77, hi: 0x77}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x32, hi: 0x32}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Embedded OpenType font", Extension: "eot",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 8, lo: 0x2, hi: 0x2}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x1, hi: 0x1}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}, {{off: 8, lo: 0x1, hi: 0x1}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}, {{off: 8, lo: 0x2, hi: 0x2}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x2, hi: 0x2}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}},
		},
		{
			Name: "TrueType Font", Extension: "ttf",
			MIME: "application/font-sfnt", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "OpenType Font", Extension: "otf",
			MIME: "application/font-sfnt", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x54, hi: 0x54}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "SQLite", Extension: "sqlite",
			MIME: "application/x-sqlite3", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x53, hi: 0x53}, {off: 1, lo: 0x51, hi: 0x51}, {off: 2, lo: 0x4c, hi: 0x4c}, {off: 3, lo: 0x69, hi: 0x69}}},
			Carver: "SQLITE",
		},
		{
			Name: "BitTorrent link", Extension: "torrent",
			MIME: "application/x-bittorrent", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x38, hi: 0x38}, {off: 2, lo: 0x3a, hi: 0x3a}, {off: 3, lo: 0x61, hi: 0x61}, {off: 4, lo: 0x6e, hi: 0x6e}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x6f, hi: 0x6f}, {off: 7, lo: 0x75, hi: 0x75}, {off: 8, lo: 0x6e, hi: 0x6e}, {off: 9, lo: 0x63, hi: 0x63}, {off: 10, lo: 0x65, hi: 0x65}, {off: 11, lo: 0x23, hi: 0x23}, {off: 12, lo: 0x23, hi: 0x23}, {off: 13, lo: 0x3a, hi: 0x3a}}, {{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x34, hi: 0x34}, {off: 2, lo: 0x3a, hi: 0x3a}, {off: 3, lo: 0x69, hi: 0x69}, {off: 4, lo: 0x6e, hi: 0x6e}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0x6f, hi: 0x6f}, {off: 7, lo: 0x64, hi: 0x64}, {off: 8, set: []byte{0x34, 0x35, 0x36}}, {off: 9, lo: 0x3a, hi: 0x3a}}},
		},
		{
			Name: "Cryptocurrency wallet", Extension: "wallet",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x1, hi: 0x1}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x62, hi: 0x62}, {off: 13, lo: 0x31, hi: 0x31}, {off: 14, lo: 0x5, hi: 0x5}, {off: 15, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Registry fragment", Extension: "hbin",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x68, hi: 0x68}, {off: 1, lo: 0x62, hi: 0x62}, {off: 2, lo: 0x69, hi: 0x69}, {off: 3, lo: 0x6e, hi: 0x6e}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Registry script", Extension: "rgs",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x48, hi: 0x48}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x43, hi: 0x43}, {off: 3, lo: 0x52, hi: 0x52}, {off: 4, lo: 0xd, hi: 0xd}, {off: 5, lo: 0xa, hi: 0xa}, {off: 6, lo: 0x5c, hi: 0x5c}, {off: 7, lo: 0x7b, hi: 0x7b}}},
		},
		{
			Name: "WinNT Registry Hive", Extension: "registry",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x72, hi: 0x72}, {off: 1, lo: 0x65, hi: 0x65}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x66, hi: 0x66}}},
		},
		{
			Name: "Windows Event Log", Extension: "evt",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x4c, hi: 0x4c}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0x4c, hi: 0x4c}, {off: 7, lo: 0x65, hi: 0x65}}},
			Carver: "EVT",
		},
		{
			Name: "Windows Event Log", Extension: "evtx",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x45, hi: 0x45}, {off: 1, lo: 0x6c, hi: 0x6c}, {off: 2, lo: 0x66, hi: 0x66}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6c, hi: 0x6c}, {off: 6, lo: 0x65, hi: 0x65}}},
			Carver: "EVTX",
		},
		{
			Name: "Windows Pagedump", Extension: "dmp",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x41, hi: 0x41}, {off: 2, lo: 0x47, hi: 0x47}, {off: 3, lo: 0x45, hi: 0x45}, {off: 4, lo: 0x44, hi: 0x44}, {off: 5, lo: 0x55, hi: 0x55}, {off: 6, set: []byte{0x4d, 0x36}}, {off: 7, set: []byte{0x50, 0x34}}}},
			Carver: "DMP",
		},
		{
			Name: "Windows Prefetch", Extension: "pf",
			MIME: "application/x-pf", Description: "",
			alts:   [][]sigCheck{{{off: 0, set: []byte{0x11, 0x17, 0x1a}}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x43, hi: 0x43}, {off: 6, lo: 0x43, hi: 0x43}, {off: 7, lo: 0x41, hi: 0x41}}},
			Carver: "PF",
		},
		{
			Name: "Windows Prefetch (Win 10)", Extension: "pf",
			MIME: "application/x-pf", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x41, hi: 0x41}, {off: 2, lo: 0x4d, hi: 0x4d}, {off: 3, lo: 0x4, hi: 0x4}, {off: 7, lo: 0x0, hi: 0x0}}},
			Carver: "PFWin10",
		},
		{
			Name: "PList (XML)", Extension: "plist",
			MIME: "application/xml", Description: "",
			alts:   [][]sigCheck{{{off: 39, lo: 0x3c, hi: 0x3c}, {off: 40, lo: 0x21, hi: 0x21}, {off: 41, lo: 0x44, hi: 0x44}, {off: 42, lo: 0x4f, hi: 0x4f}, {off: 43, lo: 0x43, hi: 0x43}, {off: 44, lo: 0x54, hi: 0x54}, {off: 45, lo: 0x59, hi: 0x59}, {off: 46, lo: 0x50, hi: 0x50}, {off: 47, lo: 0x45, hi: 0x45}, {off: 48, lo: 0x20, hi: 0x20}, {off: 49, lo: 0x70, hi: 0x70}, {off: 50, lo: 0x6c, hi: 0x6c}, {off: 51, lo: 0x69, hi: 0x69}, {off: 52, lo: 0x73, hi: 0x73}, {off: 53, lo: 0x74, hi: 0x74}}},
			Carver: "PListXML",
		},
		{
			Name: "PList (binary)", Extension: "bplist,plist,ipmeta,abcdp,mdbackup,mdinfo,strings,nib,ichat,qtz,webbookmark,webhistory",
			MIME: "application/x-plist", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x62, hi: 0x62}, {off: 1, lo: 0x70, hi: 0x70}, {off: 2, lo: 0x6c, hi: 0x6c}, {off: 3, lo: 0x69, hi: 0x69}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x30, hi: 0x30}, {off: 7, lo: 0x30, hi: 0x30}}},
		},
		{
			Name: "MacOS X Keychain", Extension: "keychain",
			MIME: "application/octet-stream", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x6b, hi: 0x6b}, {off: 1, lo: 0x79, hi: 0x79}, {off: 2, lo: 0x63, hi: 0x63}, {off: 3, lo: 0x68, hi: 0x68}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}}},
			Carver: "MacOSXKeychain",
		},
		{
			Name: "TCP Packet", Extension: "tcp",
			MIME: "application/tcp", Description: "",
			alts: [][]sigCheck{{{off: 12, lo: 0x8, hi: 0x8}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x45, hi: 0x45}, {off: 15, lo: 0x0, hi: 0x0}, {off: 21, lo: 0x0, hi: 0x0}, {off: 22, lo: 0x1, hi: 0x80}, {off: 23, lo: 0x6, hi: 0x6}}},
		},
		{
			Name: "UDP Packet", Extension: "udp",
			MIME: "application/udp", Description: "",
			alts: [][]sigCheck{{{off: 12, lo: 0x8, hi: 0x8}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x45, hi: 0x45}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, set: []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}}, {off: 22, lo: 0x1, hi: 0x80}, {off: 23, lo: 0x11, hi: 0x11}}},
		},
		{
			Name: "Compiled HTML", Extension: "chm,chw,chi",
			MIME: "application/vnd.ms-htmlhelp", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x53, hi: 0x53}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x3, hi: 0x3}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Windows Password", Extension: "pwl",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xe3, hi: 0xe3}, {off: 1, lo: 0x82, hi: 0x82}, {off: 2, lo: 0x85, hi: 0x85}, {off: 3, lo: 0x96, hi: 0x96}}},
		},
		{
			Name: "Bitlocker recovery key", Extension: "bitlocker",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x74, hi: 0x74}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x4c, hi: 0x4c}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x6f, hi: 0x6f}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x63, hi: 0x63}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x6b, hi: 0x6b}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x65, hi: 0x65}, {off: 17, lo: 0x0, hi: 0x0}, {off: 18, lo: 0x72, hi: 0x72}, {off: 19, lo: 0x0, hi: 0x0}, {off: 20, lo: 0x20, hi: 0x20}, {off: 21, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Certificate", Extension: "cer,cat,p7b,p7c,p7m,p7s,swz,rsa,crl,crt,der",
			MIME: "application/pkix-cert", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x82, hi: 0x82}, {off: 4, set: []byte{0x6, 0xa, 0x30}}}},
		},
		{
			Name: "Certificate", Extension: "cat,swz,p7m",
			MIME: "application/vnd.ms-pki.seccat", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x83, hi: 0x83}, {off: 2, lo: 0x1, hi: 0xff}, {off: 5, lo: 0x6, hi: 0x6}, {off: 6, lo: 0x9, hi: 0x9}}},
		},
		{
			Name: "PGP pubring", Extension: "pkr,gpg",
			MIME: "application/pgp-keys", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x99, hi: 0x99}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, set: []byte{0xd, 0xa2}}, {off: 3, lo: 0x4, hi: 0x4}}},
		},
		{
			Name: "PGP secring", Extension: "skr",
			MIME: "application/pgp-keys", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0xcf, hi: 0xcf}, {off: 3, lo: 0x4, hi: 0x4}}, {{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x3, hi: 0x3}, {off: 2, lo: 0xc6, hi: 0xc6}, {off: 3, lo: 0x4, hi: 0x4}}, {{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x5, hi: 0x5}, {off: 2, lo: 0x86, hi: 0x86}, {off: 3, lo: 0x4, hi: 0x4}}},
		},
		{
			Name: "PGP Safe", Extension: "pgd",
			MIME: "application/pgp-keys", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x47, hi: 0x47}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x64, hi: 0x64}, {off: 4, lo: 0x4d, hi: 0x4d}, {off: 5, lo: 0x41, hi: 0x41}, {off: 6, lo: 0x49, hi: 0x49}, {off: 7, lo: 0x4e, hi: 0x4e}, {off: 8, lo: 0x60, hi: 0x60}, {off: 9, lo: 0x1, hi: 0x1}, {off: 10, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Task Scheduler", Extension: "job",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, set: []byte{0x0, 0x1, 0x2, 0x3}}, {off: 1, set: []byte{0x5, 0x6}}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0x0, hi: 0x0}, {off: 20, lo: 0x46, hi: 0x46}, {off: 21, lo: 0x0, hi: 0x0}}},
		},
		{
			Name: "Windows Shortcut", Extension: "lnk",
			MIME: "application/x-ms-shortcut", Description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4c, hi: 0x4c}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x1, hi: 0x1}, {off: 5, lo: 0x14, hi: 0x14}, {off: 6, lo: 0x2, hi: 0x2}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0xc0, hi: 0xc0}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x0, hi: 0x0}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x0, hi: 0x0}, {off: 17, lo: 0x0, hi: 0x0}, {off: 18, lo: 0x0, hi: 0x0}, {off: 19, lo: 0x46, hi: 0x46}}},
			Carver: "LNK",
		},
		{
			Name: "Bash", Extension: "bash",
			MIME: "application/bash", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x62, hi: 0x62}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x61, hi: 0x61}, {off: 9, lo: 0x73, hi: 0x73}, {off: 10, lo: 0x68, hi: 0x68}}},
		},
		{
			Name: "Shell", Extension: "sh",
			MIME: "application/sh", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x62, hi: 0x62}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x73, hi: 0x73}, {off: 8, lo: 0x68, hi: 0x68}}},
		},
		{
			Name: "Python", Extension: "py,pyc,pyd,pyo,pyw,pyz",
			MIME: "application/python", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x70, hi: 0x70}, {off: 12, lo: 0x79, hi: 0x79}, {off: 13, lo: 0x74, hi: 0x74}, {off: 14, lo: 0x68, hi: 0x68}, {off: 15, lo: 0x6f, hi: 0x6f}, {off: 16, lo: 0x6e, hi: 0x6e}, {off: 17, set: []byte{0x32, 0x33, 0xa, 0xd}}}},
		},
		{
			Name: "Ruby", Extension: "rb",
			MIME: "application/ruby", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x72, hi: 0x72}, {off: 12, lo: 0x75, hi: 0x75}, {off: 13, lo: 0x62, hi: 0x62}, {off: 14, lo: 0x79, hi: 0x79}}},
		},
		{
			Name: "perl", Extension: "pl,pm,t,pod",
			MIME: "application/perl", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x70, hi: 0x70}, {off: 12, lo: 0x65, hi: 0x65}, {off: 13, lo: 0x72, hi: 0x72}, {off: 14, lo: 0x6c, hi: 0x6c}}},
		},
		{
			Name: "php", Extension: "php,phtml,php3,php4,php5,php7,phps,php-s,pht,phar",
			MIME: "application/php", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x3c, hi: 0x3c}, {off: 1, lo: 0x3f, hi: 0x3f}, {off: 2, lo: 0x70, hi: 0x70}, {off: 3, lo: 0x68, hi: 0x68}, {off: 4, lo: 0x70, hi: 0x70}}},
		},
		{
			Name: "Smile", Extension: "sml",
			MIME: "\tapplication/x-jackson-smile", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x3a, hi: 0x3a}, {off: 1, lo: 0x29, hi: 0x29}, {off: 2, lo: 0xa, hi: 0xa}}},
		},
		{
			Name: "Lua Bytecode", Extension: "luac",
			MIME: "application/x-lua", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1b, hi: 0x1b}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x75, hi: 0x75}, {off: 3, lo: 0x61, hi: 0x61}}},
		},
		{
			Name: "WebAssembly binary", Extension: "wasm",
			MIME: "application/octet-stream", Description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x6d, hi: 0x6d}}},
		},
	}},
}
