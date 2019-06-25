#!/bin/bash
# Copyright Ronoaldo <ronoaldo@gmail.com> (C) 2018

# Shell script to interact with ADB and grab images from SWGoH mobile game,
# process them and build a gallery style image set.

set -e

cut_them_all() {
	for image in image-[0-9][0-9][0-9].png ; do
		aux="${image%.png}"
		res="$(identify -format '%[fx:w]x%[fx:h]' $image)"
		echo -n "Processing $image ($res) ..."
		case $res in
			"1920x1080")
				if [ ! -f $aux-char.png ] ; then
					echo -n " Extracting character image... "
					convert -crop 683x924+70+138 $image $aux-char.png
				fi
				if [ ! -f $aux-portrait.png ] ; then
					echo -n " Extracting portrait image... "
					convert -crop 100x100+142+16 $image $aux-portrait.png
				fi
				if [ ! -f $aux-char-transparent.png ] ; then
					# Source: http://www.imagemagick.org/discourse-server/viewtopic.php?f=1&t=15584&start=0
					echo -n " Extracting transparent character image... "
					convert $aux-char.png image-000-char.png \
						\( -clone 0 -clone 1 -compose difference -composite -threshold 0 \) \
						-delete 1 -alpha off -compose copy_opacity -composite -fuzz 2% $aux-char-transparent.png
				fi
			;;
			"2160x1080")
				if [ ! -f $aux-char.png ] ; then
					echo -n " Extracting character image... "
					convert -crop 683x924+206+162 $image $aux-char.png
				fi
				if [ ! -f $aux-portrait.png ] ; then
					echo -n " Extracting portrait image... "
					convert -crop 100x100+142+16 $image $aux-portrait.png
				fi
				if [ ! -f $aux-char-transparent.png ] ; then
					# Source: http://www.imagemagick.org/discourse-server/viewtopic.php?f=1&t=15584&start=0
					echo -n " Extracting transparent character image... "
					convert $aux-char.png image-000-char.png \
						\( -clone 0 -clone 1 -compose difference -composite -threshold 0 \) \
						-delete 1 -alpha off -compose copy_opacity -composite -fuzz 2% $aux-char-transparent.png
				fi
			;;
			*)
				echo "Size not configured $res"
			;;
		esac
		echo
	done
}

find_names() {
	for image in image-[0-9][0-9][0-9].png ; do
		res="$(identify -format '%[fx:w]x%[fx:h]' $image)"
		case $res in
			"1920x1080"|"2160x1080")
				convert -crop 910x53+253+24 -threshold 70% $image - |\
					convert - -resize 800x600 /tmp/$image.name.pnm
			;;
			*)
				echo >&2 "Size not configured $res"
				continue
			;;
		esac
		if [ x"$image" = x"image-000.png" ] ; then
			echo "background,image-000.png"
			continue
		fi
		name="$(tesseract /tmp/$image.name.pnm stdout -l eng -psm 7 |\
			sed -e 's/Asaii Venfress/Asajj Ventress/g' \
				-e 's/Darklighfer/Darklighter/g' \
				-e 's/cc-2224/CC-2224/g' \
				-e 's/imwe/Îmwe/g' \
				-e 's/Ahsokq/Ahsoka/g' \
				-e 's/anan Jarrus/Kanan Jarrus/g' \
				-e 's/Baffle/Battle/g' \
				-e 's/Bo ba/Boba/g' \
				-e 's/BodhiRook/Bodhi Rook/g' \
				-e 's/CT-2I -0408/CT-21-0408/g' \
				-e 's/Dathc ha/Dathcha/g' \
				-e 's/KoIh/Koth/g' \
				-e 's/Th rawn/Thrawn/g' \
				-e 's/IG- 1 00/IG-100/g' \
				-e 's/^l/I/g' \
				-e 's/Jango Fe\"/Jango Fett/g' \
				-e 's/Ma rr/Marr/g' \
				-e 's/BI Battle/B1 Battle/g' \
				-e 's/Nightsisler/Nightsister/g' )"
		echo "$name,$image"
	done
}

build_index() {
	echo "<!doctype html>"
	echo "<head>"
	echo '<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" integrity="sha384-BVYiiSIFeK1dGmJRAkycuHAHRg32OmUcww7on3RYdg4Va+PmSTsz/K68vbdEjh4u" crossorigin="anonymous">'
	echo '<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css" integrity="sha384-rHyoN1iRsVXV4nD0JutlnGaslCJuC7uwjduW9SVrLvRYooPp2bWYgmgJQIXwl/Sp" crossorigin="anonymous">'
	echo '<body>'
	echo "<div class='container'>"
	echo "<h1>Star Wars Galaxy of Print Screens!</h1>"
	echo "<div class='row'>"
	i=0
	while read line ; do
		i=$((i+1))
		echo "$line" >&2
		image="$(echo $line | cut -f 2 -d,)"
		name="$(echo $line | cut -f 1 -d,)"
		aux="${image%.png}"
		portrait="${aux}-portrait.png"
		char="${aux}-char.png"
		chartransp="${aux}-char-transparent.png"
		echo "<div class='col-sm-4 col-md-4'><div class='thumbnail'>"
		echo " <img src='${char}' alt='${char}'>"
		echo " <div class='media'>"
		echo "   <div class='media-left'><img class='ml-3' src='${portrait}' alt='${portrait}'></div>"
		echo "   <div class='media-right'><h5 class='mt-0 mb-1'>${name}</h5>"
		echo "    <p>"
		echo "    <a target='_blank' href='${char}'>Character</a> - "
		echo "    <a target='_blank' href='${portrait}'>Portrait</a></p>"
		echo "    <a target='_blank' href='${chartransp}'>Transparent</a></p>"
		echo "   </div>"
		echo " </div>"
		echo "</div></div>"
		if [ $((i % 3))	-eq 0 ] ; then echo "</div><div class='row'>" ; fi
	done
	echo "</div></div>"
	echo '<script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js" integrity="sha384-Tc5IQib027qvyjSMfHjOMaLkfuWVxZxUPnCJA7l2mCWNIpG9mGCD8wGNIcPD7Txa" crossorigin="anonymous"></script>'
	echo "</body>"
}

build_assets_folder() {
	mkdir -p /tmp/assets
	while read line ; do
		image="$(echo $line | cut -f 2 -d,)"
		name="$(echo $line | cut -f 1 -d,)"
		src="${image%.png}"
		portrait="${src}-portrait.png"
		dst_portrait="${name}_portrait.png"
		char="${src}-char.png"
		dst_char="${name}.png"
		chartransp="${src}-char-transparent.png"
		dst_chartransp="${name}_transparent.png"
		cp -uv "$portrait" "/tmp/assets/$dst_portrait"
		cp -uv "$char" "/tmp/assets/$dst_char"
		cp -uv "$chartransp" "/tmp/assets/$dst_chartransp"
	done
}

screen_capture() {
	echo "Launch game, select the last character and go to basic mods. Once there, press enter"
	read -p "Press enter to begin"

	echo "Initializing ..."
	export END=160
	export i=1

	while [ $i -le $END ] ; do
		echo "Moving to next character ..."
		adb shell input tap 1887 558
		echo "Waiting for the right pose ..."
		sleep 15.5
		echo "Processing image $i ..."
		adb exec-out screencap -p > `printf "image-%03d.png" $i`
		export i=$((i+1))
	done
}

case $1 in
	--cut)  cut_them_all ;;
	--ocr)  find_names | tee index.csv  ;;
	--html)  sort index.csv | build_index | tee index.html ;;
	--assets)  sort index.csv | build_assets_folder ;;
	--adb)   screen_capture ;;
	--publish)
		gsutil -m rsync ./ gs://ronoaldo/swgoh/characters/ 
		gsutil -m acl ch -u AllUsers:R gs://ronoaldo/swgoh/characters/*
	;;
	*)
		echo "Usage of $0"
		echo "$0 --adb : using your mobile device, capture the best pose of each character"
		echo "$0 --cut : cut the images and build *-char.png and *.portrait.png"
		echo "$0 --ocr : run tesseract OCR and build index.csv with character names -> image mapping"
		echo "$0 --html: build a fancy image gallery index"
	;;
esac

