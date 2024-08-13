# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

clear
echo "${CYAN}========================================================"
echo "${BOLD}       Wii Message Board Letterhead Generator${RESET}"
echo "${CYAN}          by Alex, based on larsenv's work"
echo "${CYAN}========================================================${RESET}"

if [ ! -f "generator/input/letter.png" ]; then
    echo "${RED}${BOLD}Error:${RESET}${BOLD}Please make a letterhead that is 512x376 and save it in this directory as letter.png.${RESET}"
    exit
fi

if [ -d "generator/letterhead.d" ]; then
    rm -rf generator/letterhead.d
fi

rm -rf generator/output/*

mkdir -p generator/letterhead.d/letter.d/img/

curl -o generator/letterhead.d/wszst-setup.txt https://transfer.notkiska.pw/Z8Cbx/wszst-setup.txt
cp generator/letterhead.d/wszst-setup.txt generator/letterhead.d/letter.d/wszst-setup.txt

# Crop the letterhead into 9 parts
echo "\n${YELLOW}Cropping letterhead...${RESET}\n"
magick generator/input/letter.png -resize 512x376\! generator/input/letter.png
magick generator/input/letter.png -crop 64x144+0+0 generator/letterhead.d/letter.d/img/my_Letter_a.tpl.png
magick generator/input/letter.png -crop 384x144+64+0 generator/letterhead.d/letter.d/img/my_Letter_b.tpl.png
magick generator/input/letter.png -crop 64x144+448+0 generator/letterhead.d/letter.d/img/my_Letter_c.tpl.png
magick generator/input/letter.png -crop 64x168+0+144 generator/letterhead.d/letter.d/img/my_Letter_d.tpl.png
magick generator/input/letter.png -crop 384x168+64+144 generator/letterhead.d/letter.d/img/my_Letter_e.tpl.png
magick generator/input/letter.png -crop 64x168+448+144 generator/letterhead.d/letter.d/img/my_Letter_f.tpl.png
magick generator/input/letter.png -crop 64x64+0+312 generator/letterhead.d/letter.d/img/my_Letter_g.tpl.png
magick generator/input/letter.png -crop 384x64+64+312 generator/letterhead.d/letter.d/img/my_Letter_h.tpl.png
magick generator/input/letter.png -crop 64x64+448+312 generator/letterhead.d/letter.d/img/my_Letter_i.tpl.png

echo "\n${YELLOW}Encoding letterhead...${RESET}\n"

# Encode the cropped images and remove the originals
generator/tools/wimgt encode generator/letterhead.d/letter.d/img/*.tpl.png -x TPL.CMPR

# Create the letter_LZ.bin
generator/tools/wszst create generator/letterhead.d/letter.d/
mv generator/letterhead.d/letter.u8 generator/letterhead.d/letter_LZ.bin

# Compress the letter_LZ.bin in LZSS
generator/tools/lzss -evn generator/letterhead.d/letter_LZ.bin generator/letterhead.d/letter_LZ.bin

echo "\n${YELLOW}Encoding thumbnail...${RESET}\n"

if [ ! -f "generator/input/thumbnail.png" ]; then
    echo "${RED}${BOLD}Error:${RESET}${BOLD}Please make a thumbnail that is 144x96 and save it in the 'input' folder as 'thumbnail.png'.${RESET}"
    exit
fi

mkdir -p generator/letterhead.d/thumbnail.d/img/
cp generator/letterhead.d/wszst-setup.txt generator/letterhead.d/thumbnail.d/wszst-setup.txt

# Resize the thumbnail and encode it
magick generator/input/thumbnail.png -resize 144x96\! generator/letterhead.d/thumbnail.d/img/my_LetterS_b.tpl.png
generator/tools/wimgt encode generator/letterhead.d/thumbnail.d/img/my_LetterS_b.tpl.png -x TPL.CMPR

# Create the thumbnail_LZ.bin
generator/tools/wszst create generator/letterhead.d/thumbnail.d/
mv generator/letterhead.d/thumbnail.u8 generator/letterhead.d/thumbnail_LZ.bin

# Compress the tumbnail_LZ.bin in LZSS
generator/tools/lzss -evn generator/letterhead.d/thumbnail_LZ.bin generator/letterhead.d/thumbnail_LZ.bin

echo "\n${YELLOW}Packing up the files into .arc...${RESET}\n"

# Remove the letter.d and thumbnail.d directories, we no longer need them as we have the .bin files
rm -rf generator/letterhead.d/letter.d/
rm -rf generator/letterhead.d/thumbnail.d/

# Create the letterhead.arc
generator/tools/wszst create generator/letterhead.d
mv generator/letterhead.u8 generator/letterhead.arc

# Remove the generator/letterhead.d directory, we no longer need it as we have the .arc file
rm -rf generator/letterhead.d

if [ ! -f "generator/letterhead.arc" ]; then
    echo "${RED}${BOLD}Error:${RESET}${BOLD}The letterhead.arc file was not created check if you have permission to write to this directory.${RESET}"
    exit
fi

rm -rf generator/output/*

echo "\n${YELLOW}Converting letterhead.arc to base64...${RESET}\n"
# Convert the letterhead.arc to base64
base64 -b 76 -i generator/letterhead.arc -o generator/output/letterhead.txt

if [ -f "generator/input/sound.wav" ]; then
    echo "\n${YELLOW}Encoding sound...${RESET}\n"
    generator/tools/sharpii BNS -to generator/input/sound.wav generator/sound.bns -m
    generator/tools/wszst x generator/letterhead.arc
    mv generator/sound.bns generator/letterhead.d/sound.bns
    generator/tools/wszst create generator/letterhead.d
    mv generator/letterhead.u8 generator/letterhead.arc
    base64 -b 76 -i generator/letterhead.arc -o generator/output/letterhead.txt
    rm -rf generator/letterhead.d
fi

echo "\n${GREEN}${BOLD}The letterhead has been successfully created!${RESET}\n"

echo "${CYAN}Do you want to keep the letterhead.arc file?${RESET} (y/n)"
read keep
if [ "$keep" = "n" ]; then
    rm -rf generator/letterhead.arc
    echo "\nThe letterhead has been removed.\n\n\n${GREEN}${BOLD}The operation has completed successfully.${RESET}"
    exit
fi

mv generator/letterhead.arc generator/output/letterhead.arc
echo "\nThe letterhead.arc file has been moved to the 'generator/output' folder.\n\n\n${GREEN}${BOLD}The operation has completed successfully.${RESET}"
