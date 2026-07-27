package ops

// Code generated from CyberChef src/core/lib/FileSignatures.mjs. DO NOT EDIT.
// Regenerate via the generator in the scratchpad when the upstream table changes.

var fileSignatures = []sigCategory{
	{name: "Images", types: []fileSig{
		{
			name: "Joint Photographic Experts Group image", extension: "jpg,jpeg,jpe,thm,mpo",
			mime: "image/jpeg", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xd8, hi: 0xd8}, {off: 2, lo: 0xff, hi: 0xff}, {off: 3, set: []byte{0xc0, 0xc4, 0xdb, 0xdd, 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe7, 0xe8, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xfe}}}},
			carver: "JPEG",
		},
		{
			name: "Graphics Interchange Format image", extension: "gif",
			mime: "image/gif", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x47, hi: 0x47}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x38, hi: 0x38}, {off: 4, set: []byte{0x37, 0x39}}, {off: 5, lo: 0x61, hi: 0x61}}},
			carver: "GIF",
		},
		{
			name: "Portable Network Graphics image", extension: "png",
			mime: "image/png", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x89, hi: 0x89}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x4e, hi: 0x4e}, {off: 3, lo: 0x47, hi: 0x47}, {off: 4, lo: 0xd, hi: 0xd}, {off: 5, lo: 0xa, hi: 0xa}, {off: 6, lo: 0x1a, hi: 0x1a}, {off: 7, lo: 0xa, hi: 0xa}}},
			carver: "PNG",
		},
		{
			name: "WEBP Image", extension: "webp",
			mime: "image/webp", description: "",
			alts:   [][]sigCheck{{{off: 8, lo: 0x57, hi: 0x57}, {off: 9, lo: 0x45, hi: 0x45}, {off: 10, lo: 0x42, hi: 0x42}, {off: 11, lo: 0x50, hi: 0x50}}},
			carver: "WEBP",
		},
		{
			name: "High Efficiency Image File Format", extension: "heic,heif",
			mime: "image/heif", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, set: []byte{0x24, 0x18}}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x68, hi: 0x68}, {off: 9, lo: 0x65, hi: 0x65}, {off: 10, lo: 0x69, hi: 0x69}, {off: 11, lo: 0x63, hi: 0x63}}},
		},
		{
			name: "Camera Image File Format", extension: "crw",
			mime: "image/x-canon-crw", description: "",
			alts: [][]sigCheck{{{off: 6, lo: 0x48, hi: 0x48}, {off: 7, lo: 0x45, hi: 0x45}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x50, hi: 0x50}, {off: 10, lo: 0x43, hi: 0x43}, {off: 11, lo: 0x43, hi: 0x43}, {off: 12, lo: 0x44, hi: 0x44}, {off: 13, lo: 0x52, hi: 0x52}}},
		},
		{
			name: "Canon CR2 raw image", extension: "cr2",
			mime: "image/x-canon-cr2", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x52, hi: 0x52}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x2a, hi: 0x2a}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x52, hi: 0x52}}},
		},
		{
			name: "Tagged Image File Format image", extension: "tif",
			mime: "image/tiff", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x2a, hi: 0x2a}}},
		},
		{
			name: "Bitmap image", extension: "bmp",
			mime: "image/bmp", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x42, hi: 0x42}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 7, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 14, set: []byte{0xc, 0x28, 0x38, 0x40, 0x6c, 0x7c}}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x0, hi: 0x0}, {off: 17, lo: 0x0, hi: 0x0}}},
			carver: "BMP",
		},
		{
			name: "JPEG Extended Range image", extension: "jxr",
			mime: "image/vnd.ms-photo", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0xbc, hi: 0xbc}}},
		},
		{
			name: "Photoshop image", extension: "psd",
			mime: "image/vnd.adobe.photoshop", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x38, hi: 0x38}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Photoshop Large Document", extension: "psb",
			mime: "application/x-photoshop", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x38, hi: 0x38}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Paint Shop Pro image", extension: "psp",
			mime: "image/psp", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x69, hi: 0x69}, {off: 3, lo: 0x6e, hi: 0x6e}, {off: 4, lo: 0x74, hi: 0x74}, {off: 5, lo: 0x20, hi: 0x20}, {off: 6, lo: 0x53, hi: 0x53}, {off: 7, lo: 0x68, hi: 0x68}, {off: 8, lo: 0x6f, hi: 0x6f}, {off: 9, lo: 0x70, hi: 0x70}, {off: 10, lo: 0x20, hi: 0x20}, {off: 11, lo: 0x50, hi: 0x50}, {off: 12, lo: 0x72, hi: 0x72}, {off: 13, lo: 0x6f, hi: 0x6f}, {off: 14, lo: 0x20, hi: 0x20}, {off: 15, lo: 0x49, hi: 0x49}, {off: 16, lo: 0x6d, hi: 0x6d}}, {{off: 0, lo: 0x7e, hi: 0x7e}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x4b, hi: 0x4b}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "The GIMP image", extension: "xcf",
			mime: "image/x-xcf", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x67, hi: 0x67}, {off: 1, lo: 0x69, hi: 0x69}, {off: 2, lo: 0x6d, hi: 0x6d}, {off: 3, lo: 0x70, hi: 0x70}, {off: 4, lo: 0x20, hi: 0x20}, {off: 5, lo: 0x78, hi: 0x78}, {off: 6, lo: 0x63, hi: 0x63}, {off: 7, lo: 0x66, hi: 0x66}, {off: 8, lo: 0x20, hi: 0x20}, {off: 9, set: []byte{0x66, 0x76}}, {off: 10, set: []byte{0x69, 0x30}}, {off: 11, set: []byte{0x6c, 0x30}}, {off: 12, set: []byte{0x65, 0x31, 0x32, 0x33}}}},
		},
		{
			name: "Icon image", extension: "ico",
			mime: "image/x-icon", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, set: []byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15}}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, set: []byte{0x10, 0x20, 0x30, 0x40, 0x80}}, {off: 7, set: []byte{0x10, 0x20, 0x30, 0x40, 0x80}}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, set: []byte{0x0, 0x1}}}},
			carver: "ICO",
		},
		{
			name: "Radiance High Dynamic Range image", extension: "hdr",
			mime: "image/vnd.radiance", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x3f, hi: 0x3f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x41, hi: 0x41}, {off: 4, lo: 0x44, hi: 0x44}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x41, hi: 0x41}, {off: 7, lo: 0x4e, hi: 0x4e}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x45, hi: 0x45}, {off: 10, lo: 0xa, hi: 0xa}}},
		},
		{
			name: "Sony ARW image", extension: "arw",
			mime: "image/x-raw", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x5, hi: 0x5}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x41, hi: 0x41}, {off: 5, lo: 0x57, hi: 0x57}, {off: 6, lo: 0x31, hi: 0x31}, {off: 7, lo: 0x2e, hi: 0x2e}}},
		},
		{
			name: "Fujifilm Raw Image", extension: "raf",
			mime: "image/x-raw", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x55, hi: 0x55}, {off: 2, lo: 0x4a, hi: 0x4a}, {off: 3, lo: 0x49, hi: 0x49}, {off: 4, lo: 0x46, hi: 0x46}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x4c, hi: 0x4c}, {off: 7, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x43, hi: 0x43}, {off: 9, lo: 0x43, hi: 0x43}, {off: 10, lo: 0x44, hi: 0x44}, {off: 11, lo: 0x2d, hi: 0x2d}, {off: 12, lo: 0x52, hi: 0x52}, {off: 13, lo: 0x41, hi: 0x41}, {off: 14, lo: 0x57, hi: 0x57}}},
		},
		{
			name: "Minolta RAW image", extension: "mrw",
			mime: "image/x-raw", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}}},
		},
		{
			name: "Adobe Bridge Thumbnail Cache", extension: "bct",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x6c, hi: 0x6c}, {off: 1, lo: 0x6e, hi: 0x6e}, {off: 2, lo: 0x62, hi: 0x62}, {off: 3, lo: 0x74, hi: 0x74}, {off: 4, lo: 0x2, hi: 0x2}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Microsoft Document Imaging", extension: "mdi",
			mime: "image/vnd.ms-modi", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x45, hi: 0x45}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x2a, hi: 0x2a}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Joint Photographic Experts Group image (under Base64)", extension: "B64",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x2f, hi: 0x2f}, {off: 1, lo: 0x39, hi: 0x39}, {off: 2, lo: 0x6a, hi: 0x6a}, {off: 3, lo: 0x2f, hi: 0x2f}, {off: 4, lo: 0x34, hi: 0x34}}},
		},
		{
			name: "Portable Network Graphics image (under Base64)", extension: "B64",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x69, hi: 0x69}, {off: 1, lo: 0x56, hi: 0x56}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x52, hi: 0x52}, {off: 5, lo: 0x77, hi: 0x77}, {off: 6, lo: 0x30, hi: 0x30}}},
		},
		{
			name: "AutoCAD Drawing", extension: "dwg,123d",
			mime: "application/acad", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, set: []byte{0x30, 0x31}}, {off: 5, set: []byte{0x30, 0x31, 0x32, 0x33, 0x34, 0x35}}, {off: 6, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "AutoCAD Drawing", extension: "dwg,dwt",
			mime: "application/acad", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x31, hi: 0x31}, {off: 5, lo: 0x38, hi: 0x38}, {off: 6, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x32, hi: 0x32}, {off: 5, lo: 0x34, hi: 0x34}, {off: 6, lo: 0x0, hi: 0x0}}, {{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x43, hi: 0x43}, {off: 2, lo: 0x31, hi: 0x31}, {off: 3, lo: 0x30, hi: 0x30}, {off: 4, lo: 0x32, hi: 0x32}, {off: 5, lo: 0x37, hi: 0x37}, {off: 6, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Targa Image", extension: "tga",
			mime: "image/x-targa", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x54, hi: 0x54}, {off: 1, lo: 0x52, hi: 0x52}, {off: 2, lo: 0x55, hi: 0x55}, {off: 3, lo: 0x45, hi: 0x45}, {off: 4, lo: 0x56, hi: 0x56}, {off: 5, lo: 0x49, hi: 0x49}, {off: 6, lo: 0x53, hi: 0x53}, {off: 7, lo: 0x49, hi: 0x49}, {off: 8, lo: 0x4f, hi: 0x4f}, {off: 9, lo: 0x4e, hi: 0x4e}, {off: 10, lo: 0x2d, hi: 0x2d}, {off: 11, lo: 0x58, hi: 0x58}, {off: 12, lo: 0x46, hi: 0x46}, {off: 13, lo: 0x49, hi: 0x49}, {off: 14, lo: 0x4c, hi: 0x4c}, {off: 15, lo: 0x45, hi: 0x45}, {off: 16, lo: 0x2e, hi: 0x2e}}},
			carver: "TARGA",
		},
	}},
	{name: "Video", types: []fileSig{
		{
			name: "Matroska Multimedia Container", extension: "mkv",
			mime: "video/x-matroska", description: "",
			alts: [][]sigCheck{{{off: 31, lo: 0x6d, hi: 0x6d}, {off: 32, lo: 0x61, hi: 0x61}, {off: 33, lo: 0x74, hi: 0x74}, {off: 34, lo: 0x72, hi: 0x72}, {off: 35, lo: 0x6f, hi: 0x6f}, {off: 36, lo: 0x73, hi: 0x73}, {off: 37, lo: 0x6b, hi: 0x6b}, {off: 38, lo: 0x61, hi: 0x61}}},
		},
		{
			name: "WEBM video", extension: "webm",
			mime: "video/webm", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1a, hi: 0x1a}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0xdf, hi: 0xdf}, {off: 3, lo: 0xa3, hi: 0xa3}}},
		},
		{
			name: "Flash MP4 video", extension: "f4v",
			mime: "video/mp4", description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, set: []byte{0x66, 0x46}}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, set: []byte{0x76, 0x56}}, {off: 11, lo: 0x20, hi: 0x20}}},
		},
		{
			name: "MPEG-4 video", extension: "mp4",
			mime: "video/mp4", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, set: []byte{0x18, 0x20}}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}}, {{off: 0, lo: 0x33, hi: 0x33}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x70, hi: 0x70}, {off: 3, lo: 0x35, hi: 0x35}}, {{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x1c, hi: 0x1c}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x6d, hi: 0x6d}, {off: 9, lo: 0x70, hi: 0x70}, {off: 10, lo: 0x34, hi: 0x34}, {off: 11, lo: 0x32, hi: 0x32}, {off: 16, lo: 0x6d, hi: 0x6d}, {off: 17, lo: 0x70, hi: 0x70}, {off: 18, lo: 0x34, hi: 0x34}, {off: 19, lo: 0x31, hi: 0x31}, {off: 20, lo: 0x6d, hi: 0x6d}, {off: 21, lo: 0x70, hi: 0x70}, {off: 22, lo: 0x34, hi: 0x34}, {off: 23, lo: 0x32, hi: 0x32}, {off: 24, lo: 0x69, hi: 0x69}, {off: 25, lo: 0x73, hi: 0x73}, {off: 26, lo: 0x6f, hi: 0x6f}, {off: 27, lo: 0x6d, hi: 0x6d}}},
		},
		{
			name: "M4V video", extension: "m4v",
			mime: "video/x-m4v", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x1c, hi: 0x1c}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x4d, hi: 0x4d}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, lo: 0x56, hi: 0x56}}},
		},
		{
			name: "Quicktime video", extension: "mov",
			mime: "video/quicktime", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x14, hi: 0x14}, {off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}}},
		},
		{
			name: "Audio Video Interleave", extension: "avi",
			mime: "video/x-msvideo", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x56, hi: 0x56}, {off: 10, lo: 0x49, hi: 0x49}}},
		},
		{
			name: "Windows Media Video", extension: "wmv",
			mime: "video/x-ms-wmv", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x26, hi: 0x26}, {off: 2, lo: 0xb2, hi: 0xb2}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x8e, hi: 0x8e}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0xcf, hi: 0xcf}, {off: 7, lo: 0x11, hi: 0x11}, {off: 8, lo: 0xa6, hi: 0xa6}, {off: 9, lo: 0xd9, hi: 0xd9}}},
		},
		{
			name: "MPEG video", extension: "mpg",
			mime: "video/mpeg", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0xba, hi: 0xba}}},
		},
		{
			name: "Flash Video", extension: "flv",
			mime: "video/x-flv", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x56, hi: 0x56}, {off: 3, lo: 0x1, hi: 0x1}}},
			carver: "FLV",
		},
		{
			name: "OGG Video", extension: "ogv,ogm,opus,ogx",
			mime: "video/ogg", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x1, hi: 0x1}, {off: 29, lo: 0x76, hi: 0x76}, {off: 30, lo: 0x69, hi: 0x69}, {off: 31, lo: 0x64, hi: 0x64}, {off: 32, lo: 0x65, hi: 0x65}, {off: 33, lo: 0x6f, hi: 0x6f}}, {{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x80, hi: 0x80}, {off: 29, lo: 0x74, hi: 0x74}, {off: 30, lo: 0x68, hi: 0x68}, {off: 31, lo: 0x65, hi: 0x65}, {off: 32, lo: 0x6f, hi: 0x6f}, {off: 33, lo: 0x72, hi: 0x72}, {off: 34, lo: 0x61, hi: 0x61}}, {{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x2, hi: 0x2}, {off: 28, lo: 0x66, hi: 0x66}, {off: 29, lo: 0x69, hi: 0x69}, {off: 30, lo: 0x73, hi: 0x73}, {off: 31, lo: 0x68, hi: 0x68}, {off: 32, lo: 0x65, hi: 0x65}, {off: 33, lo: 0x61, hi: 0x61}, {off: 34, lo: 0x64, hi: 0x64}}},
		},
	}},
	{name: "Audio", types: []fileSig{
		{
			name: "Waveform Audio", extension: "wav",
			mime: "audio/x-wav", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x49, hi: 0x49}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 8, lo: 0x57, hi: 0x57}, {off: 9, lo: 0x41, hi: 0x41}, {off: 10, lo: 0x56, hi: 0x56}, {off: 11, lo: 0x45, hi: 0x45}}},
			carver: "WAV",
		},
		{
			name: "OGG audio", extension: "ogg",
			mime: "audio/ogg", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x67, hi: 0x67}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x53, hi: 0x53}}},
		},
		{
			name: "Musical Instrument Digital Interface audio", extension: "midi",
			mime: "audio/midi", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x68, hi: 0x68}, {off: 3, lo: 0x64, hi: 0x64}}},
		},
		{
			name: "MPEG-3 audio", extension: "mp3",
			mime: "audio/mpeg", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x44, hi: 0x44}, {off: 2, lo: 0x33, hi: 0x33}}, {{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfb, hi: 0xfb}}},
			carver: "MP3",
		},
		{
			name: "MPEG-4 Part 14 audio", extension: "m4a",
			mime: "audio/m4a", description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0x66, hi: 0x66}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x79, hi: 0x79}, {off: 7, lo: 0x70, hi: 0x70}, {off: 8, lo: 0x4d, hi: 0x4d}, {off: 9, lo: 0x34, hi: 0x34}, {off: 10, lo: 0x41, hi: 0x41}}, {{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x34, hi: 0x34}, {off: 2, lo: 0x41, hi: 0x41}, {off: 3, lo: 0x20, hi: 0x20}}},
		},
		{
			name: "Free Lossless Audio Codec", extension: "flac",
			mime: "audio/x-flac", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x66, hi: 0x66}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x61, hi: 0x61}, {off: 3, lo: 0x43, hi: 0x43}}},
		},
		{
			name: "Adaptive Multi-Rate audio codec", extension: "amr",
			mime: "audio/amr", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x41, hi: 0x41}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 4, lo: 0x52, hi: 0x52}, {off: 5, lo: 0xa, hi: 0xa}}},
		},
		{
			name: "Audacity", extension: "au",
			mime: "audio/x-au", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x6e, hi: 0x6e}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x2e, hi: 0x2e}, {off: 24, lo: 0x41, hi: 0x41}, {off: 25, lo: 0x75, hi: 0x75}, {off: 26, lo: 0x64, hi: 0x64}, {off: 27, lo: 0x61, hi: 0x61}, {off: 28, lo: 0x63, hi: 0x63}, {off: 29, lo: 0x69, hi: 0x69}, {off: 30, lo: 0x74, hi: 0x74}, {off: 31, lo: 0x79, hi: 0x79}, {off: 32, lo: 0x42, hi: 0x42}, {off: 33, lo: 0x6c, hi: 0x6c}, {off: 34, lo: 0x6f, hi: 0x6f}, {off: 35, lo: 0x63, hi: 0x63}, {off: 36, lo: 0x6b, hi: 0x6b}, {off: 37, lo: 0x46, hi: 0x46}, {off: 38, lo: 0x69, hi: 0x69}, {off: 39, lo: 0x6c, hi: 0x6c}, {off: 40, lo: 0x65, hi: 0x65}}},
		},
		{
			name: "Audacity Block", extension: "auf",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x75, hi: 0x75}, {off: 2, lo: 0x64, hi: 0x64}, {off: 3, lo: 0x61, hi: 0x61}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x69, hi: 0x69}, {off: 6, lo: 0x74, hi: 0x74}, {off: 7, lo: 0x79, hi: 0x79}, {off: 8, lo: 0x42, hi: 0x42}, {off: 9, lo: 0x6c, hi: 0x6c}, {off: 10, lo: 0x6f, hi: 0x6f}, {off: 11, lo: 0x63, hi: 0x63}, {off: 12, lo: 0x6b, hi: 0x6b}, {off: 13, lo: 0x46, hi: 0x46}, {off: 14, lo: 0x69, hi: 0x69}, {off: 15, lo: 0x6c, hi: 0x6c}, {off: 16, lo: 0x65, hi: 0x65}}},
		},
		{
			name: "Audio Interchange File", extension: "aif",
			mime: "audio/x-aiff", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x49, hi: 0x49}, {off: 10, lo: 0x46, hi: 0x46}, {off: 11, lo: 0x46, hi: 0x46}}},
		},
		{
			name: "Audio Interchange File (compressed)", extension: "aifc",
			mime: "audio/x-aifc", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x46, hi: 0x46}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x52, hi: 0x52}, {off: 3, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0x41, hi: 0x41}, {off: 9, lo: 0x49, hi: 0x49}, {off: 10, lo: 0x46, hi: 0x46}, {off: 11, lo: 0x43, hi: 0x43}}},
		},
	}},
	{name: "Documents", types: []fileSig{
		{
			name: "Portable Document Format", extension: "pdf",
			mime: "application/pdf", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x50, hi: 0x50}, {off: 2, lo: 0x44, hi: 0x44}, {off: 3, lo: 0x46, hi: 0x46}}},
			carver: "PDF",
		},
		{
			name: "Portable Document Format (under Base64)", extension: "B64",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x4a, hi: 0x4a}, {off: 2, lo: 0x56, hi: 0x56}, {off: 3, lo: 0x42, hi: 0x42}, {off: 4, lo: 0x45, hi: 0x45}, {off: 5, lo: 0x52, hi: 0x52}, {off: 6, lo: 0x69, hi: 0x69}}},
		},
		{
			name: "Adobe PostScript", extension: "ps,eps,ai,pfa",
			mime: "application/postscript", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x53, hi: 0x53}, {off: 4, lo: 0x2d, hi: 0x2d}, {off: 5, lo: 0x41, hi: 0x41}, {off: 6, lo: 0x64, hi: 0x64}, {off: 7, lo: 0x6f, hi: 0x6f}, {off: 8, lo: 0x62, hi: 0x62}, {off: 9, lo: 0x65, hi: 0x65}}},
		},
		{
			name: "PostScript", extension: "ps",
			mime: "application/postscript", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x25, hi: 0x25}, {off: 1, lo: 0x21, hi: 0x21}}},
		},
		{
			name: "Encapsulated PostScript", extension: "eps,ai",
			mime: "application/eps", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xc5, hi: 0xc5}, {off: 1, lo: 0xd0, hi: 0xd0}, {off: 2, lo: 0xd3, hi: 0xd3}, {off: 3, lo: 0xc6, hi: 0xc6}}},
		},
		{
			name: "Rich Text Format", extension: "rtf",
			mime: "application/rtf", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x7b, hi: 0x7b}, {off: 1, lo: 0x5c, hi: 0x5c}, {off: 2, lo: 0x72, hi: 0x72}, {off: 3, lo: 0x74, hi: 0x74}}},
			carver: "RTF",
		},
		{
			name: "Microsoft Office document/OLE2", extension: "ole2,doc,xls,dot,ppt,xla,ppa,pps,pot,msi,sdw,db,vsd,msg",
			mime: "application/msword,application/vnd.ms-excel,application/vnd.ms-powerpoint", description: "Microsoft Office documents",
			alts: [][]sigCheck{{{off: 0, lo: 0xd0, hi: 0xd0}, {off: 1, lo: 0xcf, hi: 0xcf}, {off: 2, lo: 0x11, hi: 0x11}, {off: 3, lo: 0xe0, hi: 0xe0}, {off: 4, lo: 0xa1, hi: 0xa1}, {off: 5, lo: 0xb1, hi: 0xb1}, {off: 6, lo: 0x1a, hi: 0x1a}, {off: 7, lo: 0xe1, hi: 0xe1}}},
		},
		{
			name: "Microsoft Office document/OLE2 (under Base64)", extension: "B64",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x4d, hi: 0x4d}, {off: 2, lo: 0x38, hi: 0x38}, {off: 3, lo: 0x52, hi: 0x52}, {off: 4, lo: 0x34, hi: 0x34}, {off: 5, lo: 0x4b, hi: 0x4b}, {off: 6, lo: 0x47, hi: 0x47}, {off: 7, lo: 0x78, hi: 0x78}}},
		},
		{
			name: "Microsoft Office 2007+ document", extension: "docx,xlsx,pptx",
			mime: "application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/vnd.openxmlformats-officedocument.presentationml.presentation", description: "",
			alts:   [][]sigCheck{{{off: 38, lo: 0x5f, hi: 0x5f}, {off: 39, lo: 0x54, hi: 0x54}, {off: 40, lo: 0x79, hi: 0x79}, {off: 41, lo: 0x70, hi: 0x70}, {off: 42, lo: 0x65, hi: 0x65}, {off: 43, lo: 0x73, hi: 0x73}, {off: 44, lo: 0x5d, hi: 0x5d}, {off: 45, lo: 0x2e, hi: 0x2e}, {off: 46, lo: 0x78, hi: 0x78}, {off: 47, lo: 0x6d, hi: 0x6d}, {off: 48, lo: 0x6c, hi: 0x6c}}},
			carver: "ZIP",
		},
		{
			name: "Microsoft Access database", extension: "mdb,mda,mde,mdt,fdb,psa",
			mime: "application/msaccess", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x61, hi: 0x61}, {off: 7, lo: 0x6e, hi: 0x6e}, {off: 8, lo: 0x64, hi: 0x64}, {off: 9, lo: 0x61, hi: 0x61}, {off: 10, lo: 0x72, hi: 0x72}, {off: 11, lo: 0x64, hi: 0x64}, {off: 12, lo: 0x20, hi: 0x20}, {off: 13, lo: 0x4a, hi: 0x4a}, {off: 14, lo: 0x65, hi: 0x65}, {off: 15, lo: 0x74, hi: 0x74}}},
		},
		{
			name: "Microsoft Access 2007+ database", extension: "accdb,accde,accda,accdu",
			mime: "application/msaccess", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x61, hi: 0x61}, {off: 7, lo: 0x6e, hi: 0x6e}, {off: 8, lo: 0x64, hi: 0x64}, {off: 9, lo: 0x61, hi: 0x61}, {off: 10, lo: 0x72, hi: 0x72}, {off: 11, lo: 0x64, hi: 0x64}, {off: 12, lo: 0x20, hi: 0x20}, {off: 13, lo: 0x41, hi: 0x41}, {off: 14, lo: 0x43, hi: 0x43}, {off: 15, lo: 0x45, hi: 0x45}, {off: 16, lo: 0x20, hi: 0x20}}},
		},
		{
			name: "Microsoft OneNote document", extension: "one",
			mime: "application/onenote", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xe4, hi: 0xe4}, {off: 1, lo: 0x52, hi: 0x52}, {off: 2, lo: 0x5c, hi: 0x5c}, {off: 3, lo: 0x7b, hi: 0x7b}, {off: 4, lo: 0x8c, hi: 0x8c}, {off: 5, lo: 0xd8, hi: 0xd8}, {off: 6, lo: 0xa7, hi: 0xa7}, {off: 7, lo: 0x4d, hi: 0x4d}, {off: 8, lo: 0xae, hi: 0xae}, {off: 9, lo: 0xb1, hi: 0xb1}, {off: 10, lo: 0x53, hi: 0x53}, {off: 11, lo: 0x78, hi: 0x78}, {off: 12, lo: 0xd0, hi: 0xd0}, {off: 13, lo: 0x29, hi: 0x29}, {off: 14, lo: 0x96, hi: 0x96}, {off: 15, lo: 0xd3, hi: 0xd3}}},
		},
		{
			name: "Outlook Express database", extension: "dbx",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xcf, hi: 0xcf}, {off: 1, lo: 0xad, hi: 0xad}, {off: 2, lo: 0x12, hi: 0x12}, {off: 3, lo: 0xfe, hi: 0xfe}, {off: 4, set: []byte{0x30, 0xc5, 0xc6, 0xc7}}, {off: 11, lo: 0x11, hi: 0x11}}},
		},
		{
			name: "Personal Storage Table (Outlook)", extension: "pst,ost,fdb,pab",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x21, hi: 0x21}, {off: 1, lo: 0x42, hi: 0x42}, {off: 2, lo: 0x44, hi: 0x44}, {off: 3, lo: 0x4e, hi: 0x4e}}},
		},
		{
			name: "Microsoft Exchange Database", extension: "edb",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 4, lo: 0xef, hi: 0xef}, {off: 5, lo: 0xcd, hi: 0xcd}, {off: 6, lo: 0xab, hi: 0xab}, {off: 7, lo: 0x89, hi: 0x89}, {off: 8, set: []byte{0x20, 0x23}}, {off: 9, lo: 0x6, hi: 0x6}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, set: []byte{0x0, 0x1}}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x0, hi: 0x0}, {off: 15, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "WordPerfect document", extension: "wpd,wp,wp5,wp6,wpp,bk!,wcm",
			mime: "application/wordperfect", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0x57, hi: 0x57}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x43, hi: 0x43}, {off: 7, set: []byte{0x0, 0x1, 0x2}}, {off: 8, lo: 0x1, hi: 0x1}, {off: 9, lo: 0xa, hi: 0xa}}},
		},
		{
			name: "EPUB e-book", extension: "epub",
			mime: "application/epub+zip", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x3, hi: 0x3}, {off: 3, lo: 0x4, hi: 0x4}, {off: 30, lo: 0x6d, hi: 0x6d}, {off: 31, lo: 0x69, hi: 0x69}, {off: 32, lo: 0x6d, hi: 0x6d}, {off: 33, lo: 0x65, hi: 0x65}, {off: 34, lo: 0x74, hi: 0x74}, {off: 35, lo: 0x79, hi: 0x79}, {off: 36, lo: 0x70, hi: 0x70}, {off: 37, lo: 0x65, hi: 0x65}, {off: 38, lo: 0x61, hi: 0x61}, {off: 39, lo: 0x70, hi: 0x70}, {off: 40, lo: 0x70, hi: 0x70}, {off: 41, lo: 0x6c, hi: 0x6c}, {off: 42, lo: 0x69, hi: 0x69}, {off: 43, lo: 0x63, hi: 0x63}, {off: 44, lo: 0x61, hi: 0x61}, {off: 45, lo: 0x74, hi: 0x74}, {off: 46, lo: 0x69, hi: 0x69}, {off: 47, lo: 0x6f, hi: 0x6f}, {off: 48, lo: 0x6e, hi: 0x6e}, {off: 49, lo: 0x2f, hi: 0x2f}, {off: 50, lo: 0x65, hi: 0x65}, {off: 51, lo: 0x70, hi: 0x70}, {off: 52, lo: 0x75, hi: 0x75}, {off: 53, lo: 0x62, hi: 0x62}, {off: 54, lo: 0x2b, hi: 0x2b}, {off: 55, lo: 0x7a, hi: 0x7a}, {off: 56, lo: 0x69, hi: 0x69}, {off: 57, lo: 0x70, hi: 0x70}}},
			carver: "ZIP",
		},
	}},
	{name: "Applications", types: []fileSig{
		{
			name: "Windows Portable Executable", extension: "exe,dll,drv,vxd,sys,ocx,vbx,com,fon,scr",
			mime: "application/vnd.microsoft.portable-executable", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x5a, hi: 0x5a}, {off: 3, set: []byte{0x0, 0x1, 0x2}}, {off: 5, set: []byte{0x0, 0x1, 0x2}}}},
			carver: "MZPE",
		},
		{
			name: "Executable and Linkable Format", extension: "elf,bin,axf,o,prx,so",
			mime: "application/x-executable", description: "Executable and Linkable Format file. No standard file extension.",
			alts:   [][]sigCheck{{{off: 0, lo: 0x7f, hi: 0x7f}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0x4c, hi: 0x4c}, {off: 3, lo: 0x46, hi: 0x46}}},
			carver: "ELF",
		},
		{
			name: "MacOS Mach-O object", extension: "dylib",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xca, hi: 0xca}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0xba, hi: 0xba}, {off: 3, lo: 0xbe, hi: 0xbe}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, set: []byte{0x1, 0x2, 0x3}}}, {{off: 0, lo: 0xce, hi: 0xce}, {off: 1, lo: 0xfa, hi: 0xfa}, {off: 2, lo: 0xed, hi: 0xed}, {off: 3, lo: 0xfe, hi: 0xfe}, {off: 4, lo: 0x7, hi: 0x7}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, set: []byte{0x1, 0x2, 0x3}}}},
			carver: "MACHO",
		},
		{
			name: "MacOS Mach-O 64-bit object", extension: "dylib",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xcf, hi: 0xcf}, {off: 1, lo: 0xfa, hi: 0xfa}, {off: 2, lo: 0xed, hi: 0xed}, {off: 3, lo: 0xfe, hi: 0xfe}}},
			carver: "MACHO",
		},
		{
			name: "Adobe Flash", extension: "swf",
			mime: "application/x-shockwave-flash", description: "",
			alts: [][]sigCheck{{{off: 0, set: []byte{0x43, 0x46}}, {off: 1, lo: 0x57, hi: 0x57}, {off: 2, lo: 0x53, hi: 0x53}}},
		},
		{
			name: "Java Class", extension: "class",
			mime: "application/java-vm", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xca, hi: 0xca}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0xba, hi: 0xba}, {off: 3, lo: 0xbe, hi: 0xbe}}},
		},
		{
			name: "Dalvik Executable", extension: "dex",
			mime: "application/octet-stream", description: "Dalvik Executable as used by Android",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x65, hi: 0x65}, {off: 2, lo: 0x78, hi: 0x78}, {off: 3, lo: 0xa, hi: 0xa}, {off: 4, lo: 0x30, hi: 0x30}, {off: 5, lo: 0x33, hi: 0x33}, {off: 6, lo: 0x35, hi: 0x35}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Google Chrome Extension", extension: "crx",
			mime: "application/crx", description: "Google Chrome extension or packaged app",
			alts: [][]sigCheck{{{off: 0, lo: 0x43, hi: 0x43}, {off: 1, lo: 0x72, hi: 0x72}, {off: 2, lo: 0x32, hi: 0x32}, {off: 3, lo: 0x34, hi: 0x34}}},
		},
	}},
	{name: "Archives", types: []fileSig{
		{
			name: "PKZIP archive", extension: "zip",
			mime: "application/zip", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, set: []byte{0x3, 0x5, 0x7}}, {off: 3, set: []byte{0x4, 0x6, 0x8}}}},
			carver: "ZIP",
		},
		{
			name: "PKZIP archive (under Base64)", extension: "B64",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x55, hi: 0x55}, {off: 1, lo: 0x45, hi: 0x45}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x44, hi: 0x44}, {off: 4, lo: 0x42, hi: 0x42}, {off: 5, lo: 0x42, hi: 0x42}}},
		},
		{
			name: "TAR archive", extension: "tar",
			mime: "application/x-tar", description: "",
			alts:   [][]sigCheck{{{off: 257, lo: 0x75, hi: 0x75}, {off: 258, lo: 0x73, hi: 0x73}, {off: 259, lo: 0x74, hi: 0x74}, {off: 260, lo: 0x61, hi: 0x61}, {off: 261, lo: 0x72, hi: 0x72}}},
			carver: "TAR",
		},
		{
			name: "Roshal Archive", extension: "rar",
			mime: "application/x-rar-compressed", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x52, hi: 0x52}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x72, hi: 0x72}, {off: 3, lo: 0x21, hi: 0x21}, {off: 4, lo: 0x1a, hi: 0x1a}, {off: 5, lo: 0x7, hi: 0x7}, {off: 6, set: []byte{0x0, 0x1}}}},
		},
		{
			name: "Gzip", extension: "gz",
			mime: "application/gzip", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x1f, hi: 0x1f}, {off: 1, lo: 0x8b, hi: 0x8b}, {off: 2, lo: 0x8, hi: 0x8}}},
			carver: "GZIP",
		},
		{
			name: "Bzip2", extension: "bz2",
			mime: "application/x-bzip2", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x42, hi: 0x42}, {off: 1, lo: 0x5a, hi: 0x5a}, {off: 2, lo: 0x68, hi: 0x68}}},
			carver: "BZIP2",
		},
		{
			name: "7zip", extension: "7z",
			mime: "application/x-7z-compressed", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x37, hi: 0x37}, {off: 1, lo: 0x7a, hi: 0x7a}, {off: 2, lo: 0xbc, hi: 0xbc}, {off: 3, lo: 0xaf, hi: 0xaf}, {off: 4, lo: 0x27, hi: 0x27}, {off: 5, lo: 0x1c, hi: 0x1c}}},
		},
		{
			name: "Zlib Deflate", extension: "zlib",
			mime: "application/x-deflate", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, set: []byte{0x1, 0x9c, 0xda, 0x5e}}}},
			carver: "Zlib",
		},
		{
			name: "xz compression", extension: "xz",
			mime: "application/x-xz", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0xfd, hi: 0xfd}, {off: 1, lo: 0x37, hi: 0x37}, {off: 2, lo: 0x7a, hi: 0x7a}, {off: 3, lo: 0x58, hi: 0x58}, {off: 4, lo: 0x5a, hi: 0x5a}, {off: 5, lo: 0x0, hi: 0x0}}},
			carver: "XZ",
		},
		{
			name: "Tarball", extension: "tar.z",
			mime: "application/x-gtar", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1f, hi: 0x1f}, {off: 1, set: []byte{0x9d, 0xa0}}}},
		},
		{
			name: "ISO disk image", extension: "iso",
			mime: "application/octet-stream", description: "ISO 9660 CD/DVD image file",
			alts: [][]sigCheck{{{off: 32769, lo: 0x43, hi: 0x43}, {off: 32770, lo: 0x44, hi: 0x44}, {off: 32771, lo: 0x30, hi: 0x30}, {off: 32772, lo: 0x30, hi: 0x30}, {off: 32773, lo: 0x31, hi: 0x31}}, {{off: 34817, lo: 0x43, hi: 0x43}, {off: 34818, lo: 0x44, hi: 0x44}, {off: 34819, lo: 0x30, hi: 0x30}, {off: 34820, lo: 0x30, hi: 0x30}, {off: 34821, lo: 0x31, hi: 0x31}}, {{off: 36865, lo: 0x43, hi: 0x43}, {off: 36866, lo: 0x44, hi: 0x44}, {off: 36867, lo: 0x30, hi: 0x30}, {off: 36868, lo: 0x30, hi: 0x30}, {off: 36869, lo: 0x31, hi: 0x31}}},
		},
		{
			name: "Virtual Machine Disk", extension: "vmdk",
			mime: "application/vmdk,application/x-virtualbox-vmdk", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4b, hi: 0x4b}, {off: 1, lo: 0x44, hi: 0x44}, {off: 2, lo: 0x4d, hi: 0x4d}, {off: 3, lo: 0x56, hi: 0x56}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Virtual Hard Drive", extension: "vhd",
			mime: "application/x-vhd", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x63, hi: 0x63}, {off: 1, lo: 0x6f, hi: 0x6f}, {off: 2, lo: 0x6e, hi: 0x6e}, {off: 3, lo: 0x65, hi: 0x65}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x69, hi: 0x69}, {off: 7, lo: 0x78, hi: 0x78}}},
		},
		{
			name: "Macintosh disk image", extension: "dmf,dmg",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0xd, hi: 0xd}, {off: 4, lo: 0x62, hi: 0x62}, {off: 5, lo: 0x62, hi: 0x62}, {off: 6, lo: 0x60, hi: 0x60}, {off: 7, lo: 0x60, hi: 0x60}}},
		},
		{
			name: "ARJ Archive", extension: "arj",
			mime: "application/x-arj-compressed", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x60, hi: 0x60}, {off: 1, lo: 0xea, hi: 0xea}, {off: 8, set: []byte{0x0, 0x10, 0x14}}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x2, hi: 0x2}}},
		},
		{
			name: "WinAce Archive", extension: "ace",
			mime: "application/x-ace-compressed", description: "",
			alts: [][]sigCheck{{{off: 7, lo: 0x2a, hi: 0x2a}, {off: 8, lo: 0x2a, hi: 0x2a}, {off: 9, lo: 0x41, hi: 0x41}, {off: 10, lo: 0x43, hi: 0x43}, {off: 11, lo: 0x45, hi: 0x45}, {off: 12, lo: 0x2a, hi: 0x2a}, {off: 13, lo: 0x2a, hi: 0x2a}}},
		},
		{
			name: "Macintosh BinHex Encoded File", extension: "hqx",
			mime: "application/mac-binhex", description: "",
			alts: [][]sigCheck{{{off: 11, lo: 0x6d, hi: 0x6d}, {off: 12, lo: 0x75, hi: 0x75}, {off: 13, lo: 0x73, hi: 0x73}, {off: 14, lo: 0x74, hi: 0x74}, {off: 15, lo: 0x20, hi: 0x20}, {off: 16, lo: 0x62, hi: 0x62}, {off: 17, lo: 0x65, hi: 0x65}, {off: 18, lo: 0x20, hi: 0x20}, {off: 19, lo: 0x63, hi: 0x63}, {off: 20, lo: 0x6f, hi: 0x6f}, {off: 21, lo: 0x6e, hi: 0x6e}, {off: 22, lo: 0x76, hi: 0x76}, {off: 23, lo: 0x65, hi: 0x65}, {off: 24, lo: 0x72, hi: 0x72}, {off: 25, lo: 0x74, hi: 0x74}, {off: 26, lo: 0x65, hi: 0x65}, {off: 27, lo: 0x64, hi: 0x64}, {off: 28, lo: 0x20, hi: 0x20}, {off: 29, lo: 0x77, hi: 0x77}, {off: 30, lo: 0x69, hi: 0x69}, {off: 31, lo: 0x74, hi: 0x74}, {off: 32, lo: 0x68, hi: 0x68}, {off: 33, lo: 0x20, hi: 0x20}, {off: 34, lo: 0x42, hi: 0x42}, {off: 35, lo: 0x69, hi: 0x69}, {off: 36, lo: 0x6e, hi: 0x6e}, {off: 37, lo: 0x48, hi: 0x48}, {off: 38, lo: 0x65, hi: 0x65}, {off: 39, lo: 0x78, hi: 0x78}}},
		},
		{
			name: "ALZip Archive", extension: "alz",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x41, hi: 0x41}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x5a, hi: 0x5a}, {off: 3, lo: 0x1, hi: 0x1}, {off: 4, lo: 0xa, hi: 0xa}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "KGB Compressed Archive", extension: "kgb",
			mime: "application/x-kgb-compressed", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4b, hi: 0x4b}, {off: 1, lo: 0x47, hi: 0x47}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x5f, hi: 0x5f}, {off: 4, lo: 0x61, hi: 0x61}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x63, hi: 0x63}, {off: 7, lo: 0x68, hi: 0x68}, {off: 8, lo: 0x20, hi: 0x20}, {off: 9, lo: 0x2d, hi: 0x2d}}},
		},
		{
			name: "Microsoft Cabinet", extension: "cab",
			mime: "vnd.ms-cab-compressed", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x53, hi: 0x53}, {off: 2, lo: 0x43, hi: 0x43}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Jar Archive", extension: "jar",
			mime: "application/java-archive", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x5f, hi: 0x5f}, {off: 1, lo: 0x27, hi: 0x27}, {off: 2, lo: 0xa8, hi: 0xa8}, {off: 3, lo: 0x89, hi: 0x89}}},
		},
		{
			name: "Jar Archive", extension: "jar",
			mime: "application/java-archive", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x3, hi: 0x3}, {off: 3, lo: 0x4, hi: 0x4}, {off: 4, lo: 0x14, hi: 0x14}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x8, hi: 0x8}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x8, hi: 0x8}, {off: 9, lo: 0x0, hi: 0x0}}},
			carver: "ZIP",
		},
		{
			name: "lzop compressed", extension: "lzop,lzo",
			mime: "application/x-lzop", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x89, hi: 0x89}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x5a, hi: 0x5a}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0xd, hi: 0xd}, {off: 6, lo: 0xa, hi: 0xa}, {off: 7, lo: 0x1a, hi: 0x1a}}},
			carver: "LZOP",
		},
		{
			name: "Linux deb package", extension: "deb",
			mime: "application/vnd.debian.binary-package", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x21, hi: 0x21}, {off: 1, lo: 0x3c, hi: 0x3c}, {off: 2, lo: 0x61, hi: 0x61}, {off: 3, lo: 0x72, hi: 0x72}, {off: 4, lo: 0x63, hi: 0x63}, {off: 5, lo: 0x68, hi: 0x68}, {off: 6, lo: 0x3e, hi: 0x3e}}},
			carver: "DEB",
		},
		{
			name: "Apple Disk Image", extension: "dmg",
			mime: "application/x-apple-diskimage", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x78, hi: 0x78}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0xd, hi: 0xd}, {off: 4, lo: 0x62, hi: 0x62}, {off: 5, lo: 0x62, hi: 0x62}, {off: 6, lo: 0x60, hi: 0x60}}},
		},
	}},
	{name: "Miscellaneous", types: []fileSig{
		{
			name: "UTF-8 text", extension: "txt",
			mime: "text/plain", description: "UTF-8 encoded Unicode byte order mark, commonly but not exclusively seen in text files.",
			alts: [][]sigCheck{{{off: 0, lo: 0xef, hi: 0xef}, {off: 1, lo: 0xbb, hi: 0xbb}, {off: 2, lo: 0xbf, hi: 0xbf}}},
		},
		{
			name: "UTF-32 LE text", extension: "utf32le",
			mime: "charset/utf32le", description: "Little-endian UTF-32 encoded Unicode byte order mark.",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "UTF-16 LE text", extension: "utf16le",
			mime: "charset/utf16le", description: "Little-endian UTF-16 encoded Unicode byte order mark.",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}}},
		},
		{
			name: "Web Open Font Format", extension: "woff",
			mime: "application/font-woff", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x77, hi: 0x77}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Web Open Font Format 2", extension: "woff2",
			mime: "application/font-woff", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x77, hi: 0x77}, {off: 1, lo: 0x4f, hi: 0x4f}, {off: 2, lo: 0x46, hi: 0x46}, {off: 3, lo: 0x32, hi: 0x32}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Embedded OpenType font", extension: "eot",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 8, lo: 0x2, hi: 0x2}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x1, hi: 0x1}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}, {{off: 8, lo: 0x1, hi: 0x1}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}, {{off: 8, lo: 0x2, hi: 0x2}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x2, hi: 0x2}, {off: 34, lo: 0x4c, hi: 0x4c}, {off: 35, lo: 0x50, hi: 0x50}}},
		},
		{
			name: "TrueType Font", extension: "ttf",
			mime: "application/font-sfnt", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "OpenType Font", extension: "otf",
			mime: "application/font-sfnt", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x4f, hi: 0x4f}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x54, hi: 0x54}, {off: 3, lo: 0x4f, hi: 0x4f}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "SQLite", extension: "sqlite",
			mime: "application/x-sqlite3", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x53, hi: 0x53}, {off: 1, lo: 0x51, hi: 0x51}, {off: 2, lo: 0x4c, hi: 0x4c}, {off: 3, lo: 0x69, hi: 0x69}}},
			carver: "SQLITE",
		},
		{
			name: "BitTorrent link", extension: "torrent",
			mime: "application/x-bittorrent", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x38, hi: 0x38}, {off: 2, lo: 0x3a, hi: 0x3a}, {off: 3, lo: 0x61, hi: 0x61}, {off: 4, lo: 0x6e, hi: 0x6e}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x6f, hi: 0x6f}, {off: 7, lo: 0x75, hi: 0x75}, {off: 8, lo: 0x6e, hi: 0x6e}, {off: 9, lo: 0x63, hi: 0x63}, {off: 10, lo: 0x65, hi: 0x65}, {off: 11, lo: 0x23, hi: 0x23}, {off: 12, lo: 0x23, hi: 0x23}, {off: 13, lo: 0x3a, hi: 0x3a}}, {{off: 0, lo: 0x64, hi: 0x64}, {off: 1, lo: 0x34, hi: 0x34}, {off: 2, lo: 0x3a, hi: 0x3a}, {off: 3, lo: 0x69, hi: 0x69}, {off: 4, lo: 0x6e, hi: 0x6e}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0x6f, hi: 0x6f}, {off: 7, lo: 0x64, hi: 0x64}, {off: 8, set: []byte{0x34, 0x35, 0x36}}, {off: 9, lo: 0x3a, hi: 0x3a}}},
		},
		{
			name: "Cryptocurrency wallet", extension: "wallet",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x1, hi: 0x1}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x62, hi: 0x62}, {off: 13, lo: 0x31, hi: 0x31}, {off: 14, lo: 0x5, hi: 0x5}, {off: 15, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Registry fragment", extension: "hbin",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x68, hi: 0x68}, {off: 1, lo: 0x62, hi: 0x62}, {off: 2, lo: 0x69, hi: 0x69}, {off: 3, lo: 0x6e, hi: 0x6e}, {off: 4, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Registry script", extension: "rgs",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x48, hi: 0x48}, {off: 1, lo: 0x4b, hi: 0x4b}, {off: 2, lo: 0x43, hi: 0x43}, {off: 3, lo: 0x52, hi: 0x52}, {off: 4, lo: 0xd, hi: 0xd}, {off: 5, lo: 0xa, hi: 0xa}, {off: 6, lo: 0x5c, hi: 0x5c}, {off: 7, lo: 0x7b, hi: 0x7b}}},
		},
		{
			name: "WinNT Registry Hive", extension: "registry",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x72, hi: 0x72}, {off: 1, lo: 0x65, hi: 0x65}, {off: 2, lo: 0x67, hi: 0x67}, {off: 3, lo: 0x66, hi: 0x66}}},
		},
		{
			name: "Windows Event Log", extension: "evt",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x4c, hi: 0x4c}, {off: 5, lo: 0x66, hi: 0x66}, {off: 6, lo: 0x4c, hi: 0x4c}, {off: 7, lo: 0x65, hi: 0x65}}},
			carver: "EVT",
		},
		{
			name: "Windows Event Log", extension: "evtx",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x45, hi: 0x45}, {off: 1, lo: 0x6c, hi: 0x6c}, {off: 2, lo: 0x66, hi: 0x66}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6c, hi: 0x6c}, {off: 6, lo: 0x65, hi: 0x65}}},
			carver: "EVTX",
		},
		{
			name: "Windows Pagedump", extension: "dmp",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x41, hi: 0x41}, {off: 2, lo: 0x47, hi: 0x47}, {off: 3, lo: 0x45, hi: 0x45}, {off: 4, lo: 0x44, hi: 0x44}, {off: 5, lo: 0x55, hi: 0x55}, {off: 6, set: []byte{0x4d, 0x36}}, {off: 7, set: []byte{0x50, 0x34}}}},
			carver: "DMP",
		},
		{
			name: "Windows Prefetch", extension: "pf",
			mime: "application/x-pf", description: "",
			alts:   [][]sigCheck{{{off: 0, set: []byte{0x11, 0x17, 0x1a}}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x53, hi: 0x53}, {off: 5, lo: 0x43, hi: 0x43}, {off: 6, lo: 0x43, hi: 0x43}, {off: 7, lo: 0x41, hi: 0x41}}},
			carver: "PF",
		},
		{
			name: "Windows Prefetch (Win 10)", extension: "pf",
			mime: "application/x-pf", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4d, hi: 0x4d}, {off: 1, lo: 0x41, hi: 0x41}, {off: 2, lo: 0x4d, hi: 0x4d}, {off: 3, lo: 0x4, hi: 0x4}, {off: 7, lo: 0x0, hi: 0x0}}},
			carver: "PFWin10",
		},
		{
			name: "PList (XML)", extension: "plist",
			mime: "application/xml", description: "",
			alts:   [][]sigCheck{{{off: 39, lo: 0x3c, hi: 0x3c}, {off: 40, lo: 0x21, hi: 0x21}, {off: 41, lo: 0x44, hi: 0x44}, {off: 42, lo: 0x4f, hi: 0x4f}, {off: 43, lo: 0x43, hi: 0x43}, {off: 44, lo: 0x54, hi: 0x54}, {off: 45, lo: 0x59, hi: 0x59}, {off: 46, lo: 0x50, hi: 0x50}, {off: 47, lo: 0x45, hi: 0x45}, {off: 48, lo: 0x20, hi: 0x20}, {off: 49, lo: 0x70, hi: 0x70}, {off: 50, lo: 0x6c, hi: 0x6c}, {off: 51, lo: 0x69, hi: 0x69}, {off: 52, lo: 0x73, hi: 0x73}, {off: 53, lo: 0x74, hi: 0x74}}},
			carver: "PListXML",
		},
		{
			name: "PList (binary)", extension: "bplist,plist,ipmeta,abcdp,mdbackup,mdinfo,strings,nib,ichat,qtz,webbookmark,webhistory",
			mime: "application/x-plist", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x62, hi: 0x62}, {off: 1, lo: 0x70, hi: 0x70}, {off: 2, lo: 0x6c, hi: 0x6c}, {off: 3, lo: 0x69, hi: 0x69}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x74, hi: 0x74}, {off: 6, lo: 0x30, hi: 0x30}, {off: 7, lo: 0x30, hi: 0x30}}},
		},
		{
			name: "MacOS X Keychain", extension: "keychain",
			mime: "application/octet-stream", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x6b, hi: 0x6b}, {off: 1, lo: 0x79, hi: 0x79}, {off: 2, lo: 0x63, hi: 0x63}, {off: 3, lo: 0x68, hi: 0x68}, {off: 4, lo: 0x0, hi: 0x0}, {off: 5, lo: 0x1, hi: 0x1}}},
			carver: "MacOSXKeychain",
		},
		{
			name: "TCP Packet", extension: "tcp",
			mime: "application/tcp", description: "",
			alts: [][]sigCheck{{{off: 12, lo: 0x8, hi: 0x8}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x45, hi: 0x45}, {off: 15, lo: 0x0, hi: 0x0}, {off: 21, lo: 0x0, hi: 0x0}, {off: 22, lo: 0x1, hi: 0x80}, {off: 23, lo: 0x6, hi: 0x6}}},
		},
		{
			name: "UDP Packet", extension: "udp",
			mime: "application/udp", description: "",
			alts: [][]sigCheck{{{off: 12, lo: 0x8, hi: 0x8}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x45, hi: 0x45}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, set: []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5}}, {off: 22, lo: 0x1, hi: 0x80}, {off: 23, lo: 0x11, hi: 0x11}}},
		},
		{
			name: "Compiled HTML", extension: "chm,chw,chi",
			mime: "application/vnd.ms-htmlhelp", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x49, hi: 0x49}, {off: 1, lo: 0x54, hi: 0x54}, {off: 2, lo: 0x53, hi: 0x53}, {off: 3, lo: 0x46, hi: 0x46}, {off: 4, lo: 0x3, hi: 0x3}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x0, hi: 0x0}, {off: 7, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Windows Password", extension: "pwl",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xe3, hi: 0xe3}, {off: 1, lo: 0x82, hi: 0x82}, {off: 2, lo: 0x85, hi: 0x85}, {off: 3, lo: 0x96, hi: 0x96}}},
		},
		{
			name: "Bitlocker recovery key", extension: "bitlocker",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0xff, hi: 0xff}, {off: 1, lo: 0xfe, hi: 0xfe}, {off: 2, lo: 0x42, hi: 0x42}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x0, hi: 0x0}, {off: 6, lo: 0x74, hi: 0x74}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x4c, hi: 0x4c}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x6f, hi: 0x6f}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0x63, hi: 0x63}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x6b, hi: 0x6b}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x65, hi: 0x65}, {off: 17, lo: 0x0, hi: 0x0}, {off: 18, lo: 0x72, hi: 0x72}, {off: 19, lo: 0x0, hi: 0x0}, {off: 20, lo: 0x20, hi: 0x20}, {off: 21, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Certificate", extension: "cer,cat,p7b,p7c,p7m,p7s,swz,rsa,crl,crt,der",
			mime: "application/pkix-cert", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x82, hi: 0x82}, {off: 4, set: []byte{0x6, 0xa, 0x30}}}},
		},
		{
			name: "Certificate", extension: "cat,swz,p7m",
			mime: "application/vnd.ms-pki.seccat", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x30, hi: 0x30}, {off: 1, lo: 0x83, hi: 0x83}, {off: 2, lo: 0x1, hi: 0xff}, {off: 5, lo: 0x6, hi: 0x6}, {off: 6, lo: 0x9, hi: 0x9}}},
		},
		{
			name: "PGP pubring", extension: "pkr,gpg",
			mime: "application/pgp-keys", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x99, hi: 0x99}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, set: []byte{0xd, 0xa2}}, {off: 3, lo: 0x4, hi: 0x4}}},
		},
		{
			name: "PGP secring", extension: "skr",
			mime: "application/pgp-keys", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x1, hi: 0x1}, {off: 2, lo: 0xcf, hi: 0xcf}, {off: 3, lo: 0x4, hi: 0x4}}, {{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x3, hi: 0x3}, {off: 2, lo: 0xc6, hi: 0xc6}, {off: 3, lo: 0x4, hi: 0x4}}, {{off: 0, lo: 0x95, hi: 0x95}, {off: 1, lo: 0x5, hi: 0x5}, {off: 2, lo: 0x86, hi: 0x86}, {off: 3, lo: 0x4, hi: 0x4}}},
		},
		{
			name: "PGP Safe", extension: "pgd",
			mime: "application/pgp-keys", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x50, hi: 0x50}, {off: 1, lo: 0x47, hi: 0x47}, {off: 2, lo: 0x50, hi: 0x50}, {off: 3, lo: 0x64, hi: 0x64}, {off: 4, lo: 0x4d, hi: 0x4d}, {off: 5, lo: 0x41, hi: 0x41}, {off: 6, lo: 0x49, hi: 0x49}, {off: 7, lo: 0x4e, hi: 0x4e}, {off: 8, lo: 0x60, hi: 0x60}, {off: 9, lo: 0x1, hi: 0x1}, {off: 10, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Task Scheduler", extension: "job",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, set: []byte{0x0, 0x1, 0x2, 0x3}}, {off: 1, set: []byte{0x5, 0x6}}, {off: 2, lo: 0x1, hi: 0x1}, {off: 3, lo: 0x0, hi: 0x0}, {off: 20, lo: 0x46, hi: 0x46}, {off: 21, lo: 0x0, hi: 0x0}}},
		},
		{
			name: "Windows Shortcut", extension: "lnk",
			mime: "application/x-ms-shortcut", description: "",
			alts:   [][]sigCheck{{{off: 0, lo: 0x4c, hi: 0x4c}, {off: 1, lo: 0x0, hi: 0x0}, {off: 2, lo: 0x0, hi: 0x0}, {off: 3, lo: 0x0, hi: 0x0}, {off: 4, lo: 0x1, hi: 0x1}, {off: 5, lo: 0x14, hi: 0x14}, {off: 6, lo: 0x2, hi: 0x2}, {off: 7, lo: 0x0, hi: 0x0}, {off: 8, lo: 0x0, hi: 0x0}, {off: 9, lo: 0x0, hi: 0x0}, {off: 10, lo: 0x0, hi: 0x0}, {off: 11, lo: 0x0, hi: 0x0}, {off: 12, lo: 0xc0, hi: 0xc0}, {off: 13, lo: 0x0, hi: 0x0}, {off: 14, lo: 0x0, hi: 0x0}, {off: 15, lo: 0x0, hi: 0x0}, {off: 16, lo: 0x0, hi: 0x0}, {off: 17, lo: 0x0, hi: 0x0}, {off: 18, lo: 0x0, hi: 0x0}, {off: 19, lo: 0x46, hi: 0x46}}},
			carver: "LNK",
		},
		{
			name: "Bash", extension: "bash",
			mime: "application/bash", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x62, hi: 0x62}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x61, hi: 0x61}, {off: 9, lo: 0x73, hi: 0x73}, {off: 10, lo: 0x68, hi: 0x68}}},
		},
		{
			name: "Shell", extension: "sh",
			mime: "application/sh", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x62, hi: 0x62}, {off: 4, lo: 0x69, hi: 0x69}, {off: 5, lo: 0x6e, hi: 0x6e}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x73, hi: 0x73}, {off: 8, lo: 0x68, hi: 0x68}}},
		},
		{
			name: "Python", extension: "py,pyc,pyd,pyo,pyw,pyz",
			mime: "application/python", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x70, hi: 0x70}, {off: 12, lo: 0x79, hi: 0x79}, {off: 13, lo: 0x74, hi: 0x74}, {off: 14, lo: 0x68, hi: 0x68}, {off: 15, lo: 0x6f, hi: 0x6f}, {off: 16, lo: 0x6e, hi: 0x6e}, {off: 17, set: []byte{0x32, 0x33, 0xa, 0xd}}}},
		},
		{
			name: "Ruby", extension: "rb",
			mime: "application/ruby", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x72, hi: 0x72}, {off: 12, lo: 0x75, hi: 0x75}, {off: 13, lo: 0x62, hi: 0x62}, {off: 14, lo: 0x79, hi: 0x79}}},
		},
		{
			name: "perl", extension: "pl,pm,t,pod",
			mime: "application/perl", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x23, hi: 0x23}, {off: 1, lo: 0x21, hi: 0x21}, {off: 2, lo: 0x2f, hi: 0x2f}, {off: 3, lo: 0x75, hi: 0x75}, {off: 4, lo: 0x73, hi: 0x73}, {off: 5, lo: 0x72, hi: 0x72}, {off: 6, lo: 0x2f, hi: 0x2f}, {off: 7, lo: 0x62, hi: 0x62}, {off: 8, lo: 0x69, hi: 0x69}, {off: 9, lo: 0x6e, hi: 0x6e}, {off: 10, lo: 0x2f, hi: 0x2f}, {off: 11, lo: 0x70, hi: 0x70}, {off: 12, lo: 0x65, hi: 0x65}, {off: 13, lo: 0x72, hi: 0x72}, {off: 14, lo: 0x6c, hi: 0x6c}}},
		},
		{
			name: "php", extension: "php,phtml,php3,php4,php5,php7,phps,php-s,pht,phar",
			mime: "application/php", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x3c, hi: 0x3c}, {off: 1, lo: 0x3f, hi: 0x3f}, {off: 2, lo: 0x70, hi: 0x70}, {off: 3, lo: 0x68, hi: 0x68}, {off: 4, lo: 0x70, hi: 0x70}}},
		},
		{
			name: "Smile", extension: "sml",
			mime: "\tapplication/x-jackson-smile", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x3a, hi: 0x3a}, {off: 1, lo: 0x29, hi: 0x29}, {off: 2, lo: 0xa, hi: 0xa}}},
		},
		{
			name: "Lua Bytecode", extension: "luac",
			mime: "application/x-lua", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x1b, hi: 0x1b}, {off: 1, lo: 0x4c, hi: 0x4c}, {off: 2, lo: 0x75, hi: 0x75}, {off: 3, lo: 0x61, hi: 0x61}}},
		},
		{
			name: "WebAssembly binary", extension: "wasm",
			mime: "application/octet-stream", description: "",
			alts: [][]sigCheck{{{off: 0, lo: 0x0, hi: 0x0}, {off: 1, lo: 0x61, hi: 0x61}, {off: 2, lo: 0x73, hi: 0x73}, {off: 3, lo: 0x6d, hi: 0x6d}}},
		},
	}},
}
