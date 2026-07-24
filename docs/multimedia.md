# Multimedia

Operations for images, audio, video and PDFs.

In CyberChef these operations preview their result in the browser (an `<img>`,
`<audio>`, `<video>` or `<iframe>` element). cchef is a command-line tool, so
there is no browser preview: the operations below **validate** the input and
pass the bytes through unchanged, which you save with the global `-o/--output`
flag or a shell redirect. Each also offers an **Output** option
(`--output-format`) to choose how the result is presented — raw bytes, a base64
`data:` URI, or an inline terminal preview.

The image-transform operations (Flip, Rotate, Invert, Image Filter, Image
Opacity) decode the image, apply the pixel operation, and re-encode to the source
format (GIF is re-encoded as PNG, as CyberChef does). CyberChef performs these
over the Jimp library; cchef reproduces Jimp's pixel maths exactly, so for
lossless formats (PNG/BMP/TIFF) the pixels are **identical** to CyberChef — only
the encoded bytes differ. Two caveats: **JPEG** output is re-encoded lossily and
is therefore an approximation, and **Rotate** by angles that are not a multiple
of 90° is approximate (CyberChef upscales first; cchef matches the output
dimensions but not every pixel). Read raw bytes with `--in-file` and save with
`-o`.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Extract EXIF | `extract-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Flip Image | `flip-image` | — |
| Image Filter | `image-filter` | — |
| Image Opacity | `image-opacity` | — |
| Invert Image | `invert-image` | — |
| Play Media | `play-media` | — |
| Remove EXIF | `remove-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Render Image | `render-image` | [Image file formats](https://wikipedia.org/wiki/Image_file_formats) |
| Render PDF | `render-pdf` | [PDF](https://wikipedia.org/wiki/PDF) |
| Rotate Image | `rotate-image` | — |

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

## Flip Image

Flips an image along its X (horizontal) or Y (vertical) axis. Pixel-identical to
CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Axis | option | `Horizontal` | `Horizontal` mirrors left–right; `Vertical` mirrors top–bottom. |

### Simple example

```bash
cchef flip-image --in-file photo.png --axis Vertical -o flipped.png
```

## Image Filter

Applies a greyscale or sepia filter to an image (Jimp's `greyscale`/`sepia`).
Pixel-identical to CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Filter type | option | `Greyscale` | `Greyscale` (Rec. 709 luminance) or `Sepia`. |

### Simple example

```bash
cchef image-filter --in-file photo.png --filter-type Sepia -o sepia.png
```

## Image Opacity

Multiplies every pixel's alpha channel by a percentage. Pixel-identical to
CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Opacity (%) | number | `100` | 0–100; the alpha of each pixel is scaled by this fraction. |

### Simple example

```bash
cchef image-opacity --in-file photo.png --opacity 50 -o faded.png
```

## Invert Image

Inverts the colours of an image (each RGB channel becomes 255−value; alpha is
unchanged). Pixel-identical to CyberChef for lossless formats.

Takes no options.

### Simple example

```bash
cchef invert-image --in-file photo.png -o inverted.png
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

## Rotate Image

Rotates an image by a number of degrees (Jimp's `rotate`). Rotations that are a
multiple of 90° are pixel-identical to CyberChef; other angles are approximate
(see the note above) — the output canvas is sized with CyberChef's expansion
formula but the interior pixels differ.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Rotation amount (degrees) | number | `90` | Positive or negative. Multiples of 90 relocate pixels exactly and resize the canvas; other angles expand the canvas and fill the corners with transparency. |

### Simple example

```bash
cchef rotate-image --in-file photo.png --rotation-amount-degrees 90 -o rotated.png
```
