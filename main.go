package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CompressImage 使用 libvips 命令行压缩图片，覆盖原文件
func CompressImage(path string) error {
	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	// 根据扩展名判断格式
	ext := strings.ToLower(filepath.Ext(path))
	tmpFile := path + ".tmp"

	var cmd *exec.Cmd
	Q := "80"

	switch ext {
	case ".jpg", ".jpeg":
		// JPEG 有损压缩，80% 质量
		cmd = exec.Command("vips", "jpegsave", path, tmpFile,
			"--Q", Q,
			"--strip", // 去掉EXIF/ICC等metadata
			"--optimize-coding",
		)
	case ".png":
		// PNG 无损压缩
		cmd = exec.Command("vips", "pngsave", path, tmpFile,
			"--strip",
			"--compression", "9", // 最高压缩比
			"--interlace", // 渐进式
			"--palette",   // 降低色位，大幅减少文件大小，但是可能影响颜色
		)
	case ".webp":
		// WebP 有损压缩
		cmd = exec.Command("vips", "webpsave", path, tmpFile,
			"--Q", Q,
			"--strip",
		)
	case ".avif":
		// AVIF 有损压缩
		cmd = exec.Command("vips", "heifsave", path, tmpFile,
			"--Q", "50",
			"--strip",
			"--compression", "av1",
		)
	case ".tif", ".tiff":
		// TIFF 压缩
		cmd = exec.Command("vips", "tiffsave", path, tmpFile,
			"--Q", Q,
			"--strip",
			"--compression", "jpeg",
		)
	case ".gif":
		// GIF 压缩
		cmd = exec.Command("vips", "gifsave", path, tmpFile,
			"--strip",
			"--interlace",
		)
	default:
		return fmt.Errorf("unsupported image format: %s", ext)
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("libvips failed: %v, output: %s", err, string(output))
	}

	// 用压缩后的文件覆盖原始文件
	if err := os.Rename(tmpFile, path); err != nil {
		return fmt.Errorf("failed to replace file: %v", err)
	}

	return nil
}

func main() {
	root := "."

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// 首先，处理可能发生的错误
		if err != nil {
			fmt.Printf("访问 %q 失败: %v\n", path, err)
			return err
		}

		// 检查当前条目是否是目录
		if d.IsDir() {
			fmt.Printf("发现目录: %s\n", path)
		} else {
			info, _ := d.Info()
			ori := info.Size()
			if err := CompressImage(path); err != nil {
				fmt.Printf("压缩 %s 失败: %v\n", path, err)
			} else {
				info, _ = os.Stat(path)
				fmt.Printf("压缩 %s 成功，压缩了%d%%\n", path, 100-int64(info.Size())*100/ori)
			}
		}

		// 返回 nil 表示继续遍历
		return nil
	})

	// 检查遍历过程是否出错
	if err != nil {
		log.Fatalf("遍历目录失败: %s", err)
	}
}
