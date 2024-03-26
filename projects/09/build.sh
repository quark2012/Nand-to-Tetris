#!/bin/bash -x

src="MysteryHouse"
tmp_dir="project9"

rm -fvr "${tmp_dir}"
mkdir -p "${tmp_dir}" "${tmp_dir}/source"

while read file; do
	cp -v "${src}/${file}.vm" "${tmp_dir}/"
	cp -v "${src}/${file}.jack" "${tmp_dir}/source/"
done << EOF
Const
Display
Item
Main
Monster
MysteryHouse
Player
Room
Sprite
Util
EOF

zip -r project9 project9
