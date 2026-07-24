# Multimedia

Operations for images, audio, video and PDFs.

In CyberChef these operations preview their result in the browser (an `<img>`,
`<audio>`, `<video>` or `<iframe>` element). cchef is a command-line tool, so
there is no browser preview: the operations below **validate** the input and
pass the bytes through unchanged, which you save with the global `-o/--output`
flag or a shell redirect. Each also offers an **Output** option
(`--output-format`) to choose how the result is presented — raw bytes, a base64
`data:` URI, or an inline terminal preview.

The image-transform operations (Blur, Crop, Resize, colour adjustments, …) decode
the image, apply the pixel operation, and re-encode to the source format (GIF is
re-encoded as PNG, as CyberChef does). CyberChef performs these over the Jimp
library; cchef reproduces Jimp's pixel maths exactly, so for lossless formats
(PNG/BMP/TIFF) the pixels are **identical** to CyberChef — only the encoded bytes
differ. Two caveats: **JPEG** output is re-encoded lossily and is therefore an
approximation, and **Rotate** by angles that are not a multiple of 90° is
approximate (CyberChef upscales first; cchef matches the output dimensions but not
every pixel). Read raw bytes with `--in-file` and save with `-o`.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Blur Image | `blur-image` | — |
| Contain Image | `contain-image` | — |
| Cover Image | `cover-image` | — |
| Crop Image | `crop-image` | — |
| Dither Image | `dither-image` | — |
| Extract EXIF | `extract-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Flip Image | `flip-image` | — |
| Image Brightness / Contrast | `image-brightness-contrast` | — |
| Image Filter | `image-filter` | — |
| Image Hue/Saturation/Lightness | `image-hue-saturation-lightness` | — |
| Image Opacity | `image-opacity` | — |
| Invert Image | `invert-image` | — |
| Normalise Image | `normalise-image` | — |
| Play Media | `play-media` | — |
| Remove EXIF | `remove-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Render Image | `render-image` | [Image file formats](https://wikipedia.org/wiki/Image_file_formats) |
| Render PDF | `render-pdf` | [PDF](https://wikipedia.org/wiki/PDF) |
| Resize Image | `resize-image` | — |
| Rotate Image | `rotate-image` | — |
| Sharpen Image | `sharpen-image` | — |

The resize/crop/contain/cover operations share a **resizing algorithm** option —
`Nearest Neighbour`, `Bilinear` (default), `Bicubic`, `Hermite` or `Bezier` — all
ported byte-for-byte from Jimp, so their pixels are identical to CyberChef for
lossless output.

## Blur Image

Blurs an image. Ported from Jimp's `blur` (a fast box blur) and `gaussian`.
Pixel-identical to CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Amount | number | `5` | Blur radius. For `Fast`, an integer 1–256. |
| Type | option | `Fast` | `Fast` (box blur) or `Gaussian` (slower, smoother). |

### Simple example

```bash
cchef blur-image --in-file photo.png --amount 4 --type Gaussian -o blurred.png
```

## Contain Image

Scales an image to fit inside a `Width`×`Height` box while preserving its aspect
ratio, letterboxing the remaining space. Ported from Jimp's `contain`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Width | number | `100` | Target box width in pixels. |
| Height | number | `100` | Target box height in pixels. |
| Horizontal align | option | `Center` | `Left`, `Center` or `Right`. |
| Vertical align | option | `Middle` | `Top`, `Middle` or `Bottom`. |
| Resizing algorithm | option | `Bilinear` | See the note above. |
| Opaque background | boolean | true | Composite the result over opaque black instead of transparency. |

### Simple example

```bash
cchef contain-image --in-file photo.png --width 200 --height 200 -o boxed.png
```

## Cover Image

Scales an image to completely fill a `Width`×`Height` box while preserving aspect
ratio, cropping the overflow. Ported from Jimp's `cover`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Width | number | `100` | Target box width in pixels. |
| Height | number | `100` | Target box height in pixels. |
| Horizontal align | option | `Center` | `Left`, `Center` or `Right`. |
| Vertical align | option | `Middle` | `Top`, `Middle` or `Bottom`. |
| Resizing algorithm | option | `Bilinear` | See the note above. |

### Simple example

```bash
cchef cover-image --in-file photo.png --width 200 --height 200 -o filled.png
```

## Crop Image

Crops an image to a rectangular region, or automatically crops a uniform border.
Ported from Jimp's `crop`/`autocrop`. An out-of-bounds region is rejected with
`Error cropping image. (…)`.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| X Position | number | `0` | Left edge of the crop region. |
| Y Position | number | `0` | Top edge of the crop region. |
| Width | number | `10` | Region width. |
| Height | number | `10` | Region height. |
| Autocrop | boolean | false | Ignore the region above and trim a uniform border matching the top-left pixel. |
| Autocrop tolerance (%) | number | `0.02` | Colour-difference tolerance for the border. |
| Only autocrop frames | boolean | true | Only crop when all four sides have a border. |
| Symmetric autocrop | boolean | false | Crop the same amount from opposite sides. |
| Autocrop keep border (px) | number | `0` | Leave this many border pixels in place. |

### Simple example

```bash
cchef crop-image --in-file photo.png --x-position 10 --y-position 10 --width 100 --height 80 -o cropped.png
```

### Complex example

Trim a solid border automatically:

```bash
cchef crop-image --in-file bordered.png --autocrop -o trimmed.png
```

## Dither Image

Applies an ordered (RGB565) dither effect. Ported from Jimp's `dither`.
Pixel-identical to CyberChef for lossless formats. Takes no options.

### Simple example

```bash
cchef dither-image --in-file photo.png -o dithered.png
```

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

## Image Brightness / Contrast

Adjusts image brightness and/or contrast. Ported from Jimp's
`brightness`/`contrast`. Pixel-identical to CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Brightness | number | `0` | −100 to 100. 0 leaves brightness unchanged. |
| Contrast | number | `0` | −100 to 100. 0 leaves contrast unchanged. |

### Simple example

```bash
cchef image-brightness-contrast --in-file photo.png --brightness 20 --contrast 15 -o adjusted.png
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

## Image Hue/Saturation/Lightness

Adjusts an image's hue, saturation and lightness in HSL space. Ported from
CyberChef's operation (Jimp's `color()` via tinycolor), pixel-identical to
CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Hue | number | `0` | −360 to 360 degrees to rotate the hue. |
| Saturation | number | `0` | −100 to 100. |
| Lightness | number | `0` | −100 to 100. |

### Simple example

```bash
cchef image-hue-saturation-lightness --in-file photo.png --hue 30 --saturation 20 -o tweaked.png
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

## Normalise Image

Stretches each colour channel to the full 0–255 range (auto-levels). Ported from
Jimp's `normalize`. Pixel-identical to CyberChef for lossless formats. Takes no
options.

### Simple example

```bash
cchef normalise-image --in-file photo.png -o normalised.png
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

## Resize Image

Resizes an image to a target width and height, optionally as a percentage or
preserving the aspect ratio. Ported from Jimp's `resize`/`scaleToFit` with all
five resampling algorithms.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Width | number | `100` | Target width (pixels, or percent when Unit type is Percent). |
| Height | number | `100` | Target height. |
| Unit type | option | `Pixels` | `Pixels` or `Percent` (of the source size). |
| Maintain aspect ratio | boolean | false | Scale to fit within Width×Height without distortion. |
| Resizing algorithm | option | `Bilinear` | See the note under the operation table. |

### Simple example

```bash
cchef resize-image --in-file photo.png --width 320 --height 240 -o small.png
```

### Complex example

Halve the size using percent units and nearest-neighbour sampling:

```bash
cchef resize-image --in-file photo.png --width 50 --height 50 --unit-type Percent --resizing-algorithm "Nearest Neighbour" -o half.png
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

## Sharpen Image

Sharpens an image with an unsharp mask. Ported from CyberChef's Sharpen Image,
which builds the mask from a Gaussian blur (Jimp's `gaussian`) and adds it back
where the local luminance difference exceeds a threshold. Pixel-identical to
CyberChef for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Radius | number | `2` | Gaussian blur radius used to build the mask. |
| Amount | number | `1` | Strength the mask is added back with. |
| Threshold | number | `10` | 0–100; only sharpen where the luminance difference is at least this percent. |

### Simple example

```bash
cchef sharpen-image --in-file photo.png --radius 2 --amount 1.5 --threshold 5 -o sharp.png
```
