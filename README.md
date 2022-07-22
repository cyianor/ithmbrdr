# Reader for iPod ithmb files

In an attempt to recover lost images from an iPod nano 3rd generation
(the almost square, slightly fat ones) I stumbled across the `ithmb` file
format. Apple saves thumbnails of pictures synchronised with an iPod
(and apparently older iPhones as well) in these files in varying resolutions.
The idea was that the screens were not capable of showing high resolutions
anyways, so lower resolution images were sufficient. Some of the highest
resolutions were also meant to be shown on a television.

Finding information about the `ithmb` format turned out to be rather tedious.
There are some closed source programs, mostly for Windows, that seem to be
to extract the image information. However, they usually added a watermark
that could only be removed with some extra payment. The only free source of
information I could find were some (old) internet forum threads, such as
[this one](https://forums.ilounge.com/threads/hacking-ithmb-file-format.110066/?__cf_chl_f_tk=QCtkUXDlnk4WFfcgwV2OrxJl5wu9b5uvkPnqre329y4-1642608435-0-gaNycGzNCH0).
The only open source project that attempts to read `ithmb` files that I could
find is [Keith's iPod Photo Reader](https://github.com/kebwi/Keiths_iPod_Photo_Reader),
which was the most helpful resource by far.

Check out the [README.txt](https://raw.githubusercontent.com/kebwi/Keiths_iPod_Photo_Reader/master/README.txt)
that Keith wrote for a lot of information on the format. It seems that he figured
most/all(?) of this out on his own, which is a great achievement.

For the iPod nano 3rd generation, that I had available here, it turns out that
the highest resolution images (`720 x 480 pixels`) were in files called
`F1067_x.ithmb` where `x` indicates the chunk, since these files are broken
apart if there are too many images in one.

The images are stored consecutively in the `ithmb` files, without any header.
Note that each image uses `720 * 480 * 2 bytes`. The colors were encoded in 
YCbCr or YUV (not sure) with a 4:2:0 chroma sampling scheme. Note that the
data is saved in the order below.

The first `720 * 480 bytes` encode the luminance for each pixel. The next
`360 * 240 bytes` encode the Cb or U chrominance (essentially as a
`360 x 240 pixel` grayscale) and the following `360 * 240 bytes` encode
the Cr or V chrominance.

After some more exploration, it seems like the last `720 * 120 bytes`
contain a version of the lower half of the picture encoded in the first
3/4-th of the file. It is encoded in YCbCr or YUV in blocks of four bytes
as `Cb Y Cr Y`. This is similar to a 4:2:2 chroma sampling scheme. However,
the weird thing is that the luminance channel seems to miss half the rows,
while the chroma information is fully available and is the same as the
lower half of the respective chroma channels encoded above. It is possible
that this chunk is just meant for padding, but it is unclear why half the
image would be saved here, especially in this vertically squished fashion.

## TODOs

- Figure out how the `Photo Database` file can be read. This might give reveal
  extra information about the images
- See if I can find more test data to try other iPod/iPhone models
- What is the best way to convert the colors to RGB? There seem to be many
  similar conversion formulas.