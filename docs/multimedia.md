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

**Split Colour Channels** is the exception to "one input, one output": it
produces three images, so it writes them into a directory given with `--out-dir`
rather than to `-o` or stdout.

The four **chart** operations (Heatmap, Hex Density, Scatter, Series) take
delimited numeric text and emit **SVG** on stdout — save it with `-o chart.svg`
and open it in any browser or image viewer. CyberChef draws these with d3 into a
fake DOM; cchef reproduces d3's scales, ticks, colour ramps and hexagonal binning
directly, so the geometry is identical. Two deliberate differences make the
output valid, safe standalone SVG: the clip path is spelled `clipPath` (the
element nodom emits, `clippath`, is not an SVG element, so the clip is silently
ignored by real renderers), and D3's internal `__data__` bindings — which carry
unescaped input into attributes — are never serialised. Upstream applies that
same fix, but only to Series chart.

> **Trailing newlines.** As in CyberChef, every record must have exactly the
> expected number of fields — and a trailing newline makes a final empty record,
> so the operation fails with `Each row must have length N.` Most files end in a
> newline, so strip it when piping one in:
>
> ```bash
> printf %s "$(cat points.txt)" | cchef scatter-chart -o chart.svg
> ```

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Add Text To Image | `add-text-to-image` | — |
| Blur Image | `blur-image` | — |
| Contain Image | `contain-image` | — |
| Convert Image Format | `convert-image-format` | [Image file formats](https://wikipedia.org/wiki/Image_file_formats) |
| Cover Image | `cover-image` | — |
| Crop Image | `crop-image` | — |
| Dither Image | `dither-image` | — |
| Extract EXIF | `extract-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Flip Image | `flip-image` | — |
| Generate Image | `generate-image` | — |
| Heatmap chart | `heatmap-chart` | [Heat map](https://wikipedia.org/wiki/Heat_map) |
| Hex Density chart | `hex-density-chart` | — |
| Image Brightness / Contrast | `image-brightness-contrast` | — |
| Image Filter | `image-filter` | — |
| Image Hue/Saturation/Lightness | `image-hue-saturation-lightness` | — |
| Image Opacity | `image-opacity` | — |
| Invert Image | `invert-image` | — |
| Normalise Image | `normalise-image` | — |
| Optical Character Recognition | `optical-character-recognition` | [Optical character recognition](https://wikipedia.org/wiki/Optical_character_recognition) |
| Play Media | `play-media` | — |
| Remove EXIF | `remove-exif` | [Exif](https://wikipedia.org/wiki/Exif) |
| Render Image | `render-image` | [Image file formats](https://wikipedia.org/wiki/Image_file_formats) |
| Render PDF | `render-pdf` | [PDF](https://wikipedia.org/wiki/PDF) |
| Resize Image | `resize-image` | — |
| Rotate Image | `rotate-image` | — |
| Scatter chart | `scatter-chart` | [Scatter plot](https://wikipedia.org/wiki/Scatter_plot) |
| Series chart | `series-chart` | — |
| Sharpen Image | `sharpen-image` | — |
| Split Colour Channels | `split-colour-channels` | [Channel (digital image)](https://wikipedia.org/wiki/Channel_(digital_image)) |

The resize/crop/contain/cover operations share a **resizing algorithm** option —
`Nearest Neighbour`, `Bilinear` (default), `Bicubic`, `Hermite` or `Bezier` — all
ported byte-for-byte from Jimp, so their pixels are identical to CyberChef for
lossless output.

## Add Text To Image

Draws text onto an image. CyberChef renders the text with Jimp from bundled 72px
Roboto **bitmap-font atlases**; cchef embeds those same atlases and reproduces
Jimp's glyph blitting, so the result is pixel-identical for lossless formats.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Text | string | `""` | The text to draw. `\n` starts a new line. |
| Horizontal align | option | `None` | `None` uses X position; otherwise `Left`, `Center` or `Right`. |
| Vertical align | option | `None` | `None` uses Y position; otherwise `Top`, `Middle` or `Bottom`. |
| X position | number | `0` | Left edge, when Horizontal align is `None`. |
| Y position | number | `0` | Top edge, when Vertical align is `None`. |
| Size | number | `32` | Point size, minimum 8. The 72px glyphs are bicubically scaled to it. |
| Font face | option | `Roboto` | `Roboto`, `Roboto Black`, `Roboto Mono` or `Roboto Slab`. |
| Red / Green / Blue / Alpha | number | `255` | Text colour, each 0–255. |

Only the four Roboto faces are available, as in CyberChef — there is no option to
supply your own font. Characters outside the atlas are drawn as `?`, and
whitespace the atlas has no glyph for is skipped.

The text is laid out into its own bitmap before being scaled and composited, and
CyberChef sizes that bitmap with Jimp's `measureTextHeight`. That call reserves
one line per space-separated word plus one more, so text with several words gets
a taller transparent block beneath it than the single drawn line needs. This is
faithful to CyberChef: it only matters if you align vertically, where the extra
height shifts `Middle` and `Bottom` upwards.

### Simple example

```bash
cchef add-text-to-image --in-file photo.png --text "Hello" -o titled.png
```

### Complex example

Centre 40pt red Roboto Slab across the middle of the image:

```bash
cchef add-text-to-image --in-file photo.png \
  --text "CONFIDENTIAL" --font-face "Roboto Slab" --size 40 \
  --horizontal-align Center --vertical-align Middle \
  --red 255 --green 0 --blue 0 -o stamped.png
```

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

## Convert Image Format

Decodes an image and re-encodes it as JPEG, PNG, BMP or TIFF.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Output Format | option | `JPEG` | `JPEG`, `PNG`, `BMP` or `TIFF`. |
| JPEG Quality | number | `80` | 1–100. Only used for JPEG output. |
| PNG Filter Type | option | `Auto` | Accepted for compatibility; see the note below. |
| PNG Deflate Level | number | `9` | 0–9. Only affects PNG compression, never the pixels. |

For the lossless targets (PNG, BMP, TIFF) the decoded pixels are identical to
CyberChef; only the encoded bytes differ, since Go and Jimp use different
encoders. BMP is 24-bit, so alpha is flattened to opaque — as it is in CyberChef.
JPEG output is lossy and re-encoded with Go's encoder, so it is an approximation.

`PNG Filter Type` and `PNG Deflate Level` do not change the image. Go's PNG
encoder exposes no per-scanline filter selector, so the filter type is accepted
and ignored, and the deflate level is mapped onto Go's four compression levels
(`0` → none, `1–3` → best speed, `4–7` → default, `8–9` → best compression).
Decoding any of these outputs gives the same pixels regardless.

### Simple example

```bash
cchef convert-image-format --in-file photo.png --output-format PNG -o photo-copy.png
```

### Complex example

Re-encode as a high-quality JPEG:

```bash
cchef convert-image-format --in-file photo.png \
  --output-format JPEG --jpeg-quality 95 -o photo.jpg
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

## Generate Image

Builds a PNG whose pixels are read straight from the input bytes — useful for
eyeballing the structure of an unknown binary. Pixel-identical to CyberChef.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Mode | option | `Greyscale` | Bytes consumed per pixel: `Greyscale` 1, `RG` 2, `RGB` 3, `RGBA` 4, `Bits` ⅛. |
| Pixel Scale Factor | number | `8` | 1–64. Each source pixel becomes an N×N block (nearest-neighbour). |
| Pixels per row | number | `64` | 1–2048. Image width, before scaling. |

In `Bits` mode each input byte becomes eight black-or-white pixels, most
significant bit first, with a set bit rendered black. Every other mode reads its
channels in order and leaves alpha opaque unless the mode supplies it. The input
length must be a whole number of pixels, or the operation fails with
`Number of bytes is not a divisor of N`.

Empty input produces a single transparent row (matching Jimp's clamping) when a
scale factor is set. With `--pixel-scale-factor 1` there is no scaling to clamp
the height, so the image would be zero pixels tall; CyberChef emits a height-0
PNG there, but Go's encoder rejects that size, so cchef reports an error instead.

### Simple example

```bash
cchef generate-image --in-file firmware.bin -o firmware.png
```

### Complex example

Render a binary one bit per pixel at 1024 bits per row, unscaled:

```bash
cchef generate-image --in-file firmware.bin \
  --mode Bits --pixels-per-row 1024 --pixel-scale-factor 1 -o bits.png
```

## Heatmap chart

Bins two-variable data into a grid and shades each cell by how many points fall
in it. Input is one `x y` record per line; output is SVG.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Record delimiter | option | `Line feed` | `Line feed` or `CRLF`. |
| Field delimiter | option | `Space` | `Space`, `Comma`, `Semi-colon`, `Colon` or `Tab`. |
| Number of vertical bins | number | `25` | Rows in the grid. Must be above 0. |
| Number of horizontal bins | number | `25` | Columns in the grid. Must be above 0. |
| Use column headers as labels | boolean | true | Take the axis labels from the first record. |
| X label | string | `""` | Used when headers are not taken as labels. |
| Y label | string | `""` | Used when headers are not taken as labels. |
| Draw bin edges | boolean | false | Outline each cell. |
| Min colour value | string | `white` | Colour for an empty cell. Any CSS colour. |
| Max colour value | string | `black` | Colour for the fullest cell. Any CSS colour. |

Cells are shaded by blending the two colours through CIELAB, as CyberChef does.
The data must vary in both axes: a column of identical x or y values is rejected,
since there would be nothing to spread across the bins.

### Simple example

```bash
printf '100 100\n200 200\n300 300' |
  cchef heatmap-chart --number-of-vertical-bins 5 --number-of-horizontal-bins 5 -o heat.svg
```

### Complex example

Read from a file, label the axes, outline the cells and shade white to red:

```bash
printf %s "$(cat points.csv)" |
  cchef heatmap-chart --field-delimiter Comma \
    --number-of-vertical-bins 20 --number-of-horizontal-bins 20 \
    --x-label "requests" --y-label "latency" --draw-bin-edges \
    --min-colour-value white --max-colour-value red -o heat.svg
```

## Hex Density chart

Groups points into hexagonal cells and shades each by how many it holds — a
scatter plot that stays readable when points overlap. Output is SVG.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Record delimiter | option | `Line feed` | `Line feed` or `CRLF`. |
| Field delimiter | option | `Space` | `Space`, `Comma`, `Semi-colon`, `Colon` or `Tab`. |
| Pack radius | number | `25` | Hexagon size used for grouping the points. |
| Draw radius | number | `15` | Hexagon size actually drawn, so cells can be spaced apart. |
| Use column headers as labels | boolean | true | Take the axis labels from the first record. |
| X label | string | `""` | Used when headers are not taken as labels. |
| Y label | string | `""` | Used when headers are not taken as labels. |
| Draw hexagon edges | boolean | false | Outline each hexagon. |
| Min colour value | string | `white` | Colour for the emptiest hexagon. Any CSS colour. |
| Max colour value | string | `black` | Colour for the fullest hexagon. Any CSS colour. |
| Draw empty hexagons within data boundaries | boolean | false | Also draw cells holding no points. |

### Simple example

```bash
printf '100 100\n200 200\n300 300' | cchef hex-density-chart -o hex.svg
```

### Complex example

Tighter packing, outlined hexagons, and the empty cells filled in:

```bash
printf %s "$(cat points.csv)" |
  cchef hex-density-chart --field-delimiter Comma \
    --pack-radius 15 --draw-radius 12 --draw-hexagon-edges \
    --draw-empty-hexagons-within-data-boundaries \
    --min-colour-value white --max-colour-value blue -o hex.svg
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

## Optical Character Recognition

Reads text out of an image.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Show confidence | boolean | true | Prefix the text with the mean word confidence. |
| OCR Engine Mode | option | `LSTM only` | `Tesseract only`, `LSTM only` or `Tesseract/LSTM Combined`. |

> **Requires `tesseract`.** This is the one operation cchef does not perform
> itself. CyberChef runs Tesseract compiled to WebAssembly in the browser; the
> only cgo-free way to do that from Go is to host the WebAssembly module, and the
> one library that did so is unmaintained and several times slower. Shelling out
> to the installed engine is what keeps cchef a single static binary with no cgo.
> Install it with `brew install tesseract` (macOS) or
> `apt install tesseract-ocr` (Debian/Ubuntu); without it the operation fails
> with a message saying so. Everything else in cchef works regardless.

Because it is the same engine and the same `eng` language data, the recognised
text matches CyberChef's. The confidence figure is the mean confidence of the
recognised words.

`Tesseract only` and `Tesseract/LSTM Combined` need the legacy engine, which
Tesseract 5 no longer ships in its default `eng.traineddata` — those modes fail
with the engine's own message unless legacy training data is installed. `LSTM
only`, the default, is unaffected.

### Simple example

```bash
cchef optical-character-recognition --in-file scan.png
```

Output:

```
Confidence: 96%

Hello World
```

### Complex example

Just the text, with no confidence line, ready to pipe onward:

```bash
cchef optical-character-recognition --in-file scan.png --show-confidence=false |
  cchef to-base64
```

Output:

```
SGVsbG8gV29ybGQK
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

## Scatter chart

Plots two-variable data as individual points. Input is one `x y` record per
line; output is SVG.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Record delimiter | option | `Line feed` | `Line feed` or `CRLF`. |
| Field delimiter | option | `Space` | `Space`, `Comma`, `Semi-colon`, `Colon` or `Tab`. |
| Use column headers as labels | boolean | true | Take the axis labels from the first record. |
| X label | string | `""` | Used when headers are not taken as labels. |
| Y label | string | `""` | Used when headers are not taken as labels. |
| Colour | string | `black` | Point colour. Any CSS colour. |
| Point radius | number | `10` | Radius of each point. |
| Use colour from third column | boolean | false | Read each point's colour from a third field. |

The axes extend a tenth of the data's span past it at each end, so points are
never drawn on the axis lines.

### Simple example

```bash
printf '100 100\n200 200\n300 300' | cchef scatter-chart --point-radius 5 -o scatter.svg
```

### Complex example

Colour each point from a third column:

```bash
printf '1,2,red\n3,4,blue\n5,9,green' |
  cchef scatter-chart --field-delimiter Comma \
    --use-colour-from-third-column --point-radius 4 \
    --x-label x --y-label y -o scatter.svg
```

## Series chart

Draws one line graph per named series over a shared x axis. Input is one
`series x value` record per line; output is SVG.

| Option | Type | Default | Notes |
| --- | --- | --- | --- |
| Record delimiter | option | `Line feed` | `Line feed` or `CRLF`. |
| Field delimiter | option | `Space` | `Space`, `Comma`, `Semi-colon`, `Colon` or `Tab`. |
| X label | string | `""` | Label under the x axis. |
| Point radius | number | `1` | Radius of the pip drawn at each value. |
| Series colours | string | `mediumseagreen, dodgerblue, tomato` | Comma-separated CSS colours, cycled across the series. |

Series and x values keep the order they first appear in, and each series gets its
own y axis scaled to its own range. A series with no value at some x simply skips
that point rather than drawing through the gap.

### Simple example

```bash
printf 'a 1 10\na 2 20\nb 1 5\nb 2 25' | cchef series-chart -o series.svg
```

### Complex example

Comma-separated input with a labelled axis and custom colours:

```bash
printf %s "$(cat metrics.csv)" |
  cchef series-chart --field-delimiter Comma \
    --x-label "week" --point-radius 3 \
    --series-colours "tomato, steelblue" -o series.svg
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

## Split Colour Channels

Splits an image into three PNGs — its red, green and blue channels, each with the
other two channels zeroed and the original alpha kept. Pixel-identical to
CyberChef.

This operation has no options. It is the one operation that produces **several
files** rather than a single output, so it writes into a directory given with
`--out-dir` instead of to stdout or `-o`:

```bash
cchef split-colour-channels --in-file photo.png --out-dir ./channels
# ./channels/red.png, ./channels/green.png, ./channels/blue.png
```

Without `--out-dir` the operation fails, since there is nowhere to put the three
files. For the same reason it cannot be chained: it has to be the last step of a
recipe.

### Complex example

Split every image in a directory. Each input's channels go into their own
subdirectory, named after the input file, so the results cannot collide:

```bash
cchef split-colour-channels --in-dir ./photos --out-dir ./channels
# ./channels/sunset/red.png, ./channels/sunset/green.png, ...
```
