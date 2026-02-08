package handler

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
	"wa-bot/internal/utils"
)

func convertToWebP(inputPath string, isVideo bool) (string, error) {
	tmpWebp := fmt.Sprintf("temp_%d.webp", time.Now().UnixNano())
	
	scaleFilter := "scale='if(gt(iw,ih),512,-2)':'if(gt(iw,ih),-2,512)'"
	var cmd *exec.Cmd

	if isVideo {
		scaleFilter += ":flags=lanczos,fps=15,setpts=PTS"
		cmd = exec.Command("ffmpeg", "-y", "-i", inputPath,
			"-vf", scaleFilter,
			"-vcodec", "libwebp", "-lossless", "0", "-compression_level", "4", "-q:v", "50",
			"-loop", "0", "-preset", "default", "-an", "-vsync", "0",
			"-ss", "00:00:00", "-t", "00:00:06",
			tmpWebp)
	} else {
		cmd = exec.Command("ffmpeg", "-y", "-i", inputPath,
			"-vf", scaleFilter,
			"-vcodec", "libwebp", "-lossless", "0", "-compression_level", "4", "-q:v", "80",
			"-loop", "0", "-an", "-vsync", "0",
			tmpWebp)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		utils.LogDebug("FFmpeg Error: " + string(output))
		return "", err
	}
	return tmpWebp, nil
}

func convertStickerToMedia(webpPath string, targetFormat string) (string, error) {
	randID := time.Now().UnixNano()
	outputFile := fmt.Sprintf("temp_%d.%s", randID, targetFormat)

	if targetFormat == "gif" {
		// WebP -> GIF (ImageMagick)
		cmd := exec.Command("convert", webpPath, "-coalesce", outputFile)
		out, err := cmd.CombinedOutput()
		if err != nil {
			utils.LogDebug("ImageMagick Error: " + string(out))
			return "", err
		}
		return outputFile, nil
	} else if targetFormat == "mp4" {
		// WebP -> GIF -> MP4
		tmpGif := fmt.Sprintf("temp_%d.gif", randID)
		defer os.Remove(tmpGif)
		
		cmd1 := exec.Command("convert", webpPath, "-coalesce", tmpGif)
		if out, err := cmd1.CombinedOutput(); err != nil {
			utils.LogDebug("Magick (Gif Step) Error: " + string(out))
			return "", err
		}

		cmd2 := exec.Command("ffmpeg", "-y", "-i", tmpGif,
			"-movflags", "faststart", "-pix_fmt", "yuv420p",
			"-c:v", "libx264", "-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
			outputFile)
			
		if out, err := cmd2.CombinedOutput(); err != nil {
			utils.LogDebug("FFmpeg (Mp4 Step) Error: " + string(out))
			return "", err
		}
		return outputFile, nil
	} else if targetFormat == "jpg" {
		// WebP -> JPG
		cmd := exec.Command("convert", fmt.Sprintf("%s[0]", webpPath), outputFile)
		if _, err := cmd.CombinedOutput(); err != nil {
			// Fallback FFmpeg
			cmdFallback := exec.Command("ffmpeg", "-y", "-i", webpPath, "-vframes", "1", outputFile)
			if out, err := cmdFallback.CombinedOutput(); err != nil {
				utils.LogDebug("Convert Img Error: " + string(out))
				return "", err
			}
		}
		return outputFile, nil
	}

	return "", fmt.Errorf("unknown format")
}
