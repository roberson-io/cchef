# Multimedia

Operations for images, audio, video and PDFs.

In CyberChef these operations preview their result in the browser (an `<img>`,
`<audio>`, `<video>` or `<iframe>` element). cchef is a command-line tool, so
there is no browser preview: the operations below **validate** the input and
pass the bytes through unchanged, which you save with the global `-o/--output`
flag or a shell redirect. Each also offers an **Output** option
(`--output-format`) to choose how the result is presented — raw bytes, a base64
`data:` URI, or an inline terminal preview.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Extract EXIF | `extract-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Play Media | `play-media` | — |
| Remove EXIF | `remove-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Render Image | `render-image` | [Image file formats](https://wikipedia.org/wiki/Image_file_formats) |
| Render PDF | `render-pdf` | [PDF](https://wikipedia.org/wiki/PDF) |

## Extract EXIF

Extracts EXIF metadata from an image (JPEG/TIFF). EXIF records information about
the image and the device that produced it. This is a from-scratch port of the
npm `exif-parser` library that CyberChef uses, reproducing its tag names, value
formatting (rationals as decimals, EXIF dates as UNIX timestamps) and GPS
coordinate conversion. The output lists a tag count followed by one `name: value`
per line. Malformed input yields `Could not extract EXIF data from image: …`.

Takes no options. The input is raw bytes, so use the global `--in-file` flag or a
decode step such as `from-hex`.

### Simple example

```bash
cchef extract-exif --in-file photo.jpg
```

Output (from a sample with GPS metadata):

```
Found 9 tags.

Make: cchef
Orientation: 1
XResolution: 72
GPSLatitudeRef: N
GPSLatitude: 51.5
GPSLongitudeRef: E
GPSLongitude: 0.125
ExposureTime: 0.01
InteropIndex: 
```

## Play Media

Validates that the input is an audio or video file (by magic bytes) and outputs
it. Ported from CyberChef's Play Media; the browser `<audio>`/`<video>` preview
is replaced by the Output option below. A non-media input is rejected.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Input format | option | `Raw` | How to read the input: `Raw`, `Base64` or `Hex`. |
| Output | option | `Raw` | `Raw` emits the media bytes; `Base64` emits a `data:<mime>;base64,…` URI. |

### Simple example

Save recovered media to a file (raw passthrough):

```bash
cchef play-media --in-file sound.wav -o recovered.wav
```

### Complex example

Emit a base64 `data:` URI for the detected media type:

```bash
cchef from-hex -i "524946460000000057415645" | cchef play-media --output-format Base64
```

Output:

```
data:audio/x-wav;base64,UklGRgAAAABXQVZF
```

## Remove EXIF

Removes the EXIF metadata segment from a JPEG image, returning the stripped
bytes. Ported byte-for-byte from CyberChef's vendored piexifjs routine: it splits
the JPEG into segments and drops the APP1 `Exif` segment. If the image has no
EXIF data it is returned unchanged; non-JPEG input is rejected with
`Could not remove EXIF data from image: Given data is not jpeg.`

Takes no options. Save the result with `-o` or a redirect.

### Simple example

Strip EXIF and confirm none remains by piping into Extract EXIF:

```bash
cchef remove-exif --in-file photo.jpg | cchef extract-exif
```

Output:

```
Found 0 tags.
```

## Render Image

Validates that the input is an image (jpg/jpeg, png, gif, webp, bmp, ico) and
outputs it. Ported from CyberChef's Render Image; the browser `<img>` preview is
replaced by the Output option below. A non-image input is rejected with
`Invalid file type`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Input format | option | `Raw` | How to read the input: `Raw`, `Base64` or `Hex`. |
| Output | option | `Raw` | `Raw` emits the image bytes; `Base64` emits a `data:<mime>;base64,…` URI; `Terminal` renders an inline preview. |

The `Terminal` output renders the image in terminals that support inline images:
iTerm2 (and WezTerm) via the iTerm2 protocol for any format, and kitty via the
kitty graphics protocol for PNG. On an unsupported terminal it reports an error.

### Simple example

Validate and save an image (raw passthrough):

```bash
cchef render-image --in-file photo.png -o out.png
```

### Complex example

Emit a base64 `data:` URI (here from a 1×1 GIF):

```bash
cchef from-hex -i "47494638396101000100" | cchef render-image --output-format Base64
```

Output:

```
data:image/gif;base64,R0lGODlhAQABAA==
```

## Render PDF

Validates that the input begins with the `%PDF` signature and outputs it. Ported
from CyberChef's Render PDF; the browser `<iframe>` preview is replaced by the
Output option below. Input that is not a PDF is rejected with
`Input does not appear to be a PDF file.`

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Input format | option | `Base64` | How to read the input: `Base64` or `Raw`. |
| Output | option | `Raw` | `Raw` emits the PDF bytes; `Base64` emits a `data:application/pdf;base64,…` URI. |

### Simple example

Validate and save a PDF (raw passthrough):

```bash
cchef render-pdf --input-format Raw --in-file report.pdf -o out.pdf
```

### Complex example

Emit a base64 `data:` URI:

```bash
cchef render-pdf --input-format Raw --output-format Base64 -i "%PDF-1.7 hi"
```

Output:

```
data:application/pdf;base64,JVBERi0xLjcgaGk=
```
