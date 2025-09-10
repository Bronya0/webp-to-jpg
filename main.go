package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
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

	log.Print("请输入本地目录路径（Windows下如 C:\\Users\\YourName\\Documents）: ")

	// 使用 bufio.Scanner 读取整行（推荐，支持带空格的路径）
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	root := scanner.Text()
	// 可选：去除首尾空格
	root = strings.TrimSpace(root)
	if root == "" {
		log.Println("❌ 路径不能为空！")
		return
	}
	log.Printf("✅ 你输入的路径是: %s\n", root)

	paths := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		// 首先，处理可能发生的错误
		if err != nil {
			log.Printf("访问 %q 失败: %v\n", path, err)
			return err
		}

		// 检查当前条目是否是目录
		if d.IsDir() {
			log.Printf("发现目录: %s\n", path)
		} else {
			// 只处理图片
			suffix := strings.ToLower(filepath.Ext(path))
			if suffix == ".jpg" || suffix == ".jpeg" || suffix == ".png" || suffix == ".webp" || suffix == ".avif" || suffix == ".tif" || suffix == ".tiff" || suffix == ".gif" {
				paths = append(paths, path)
			}
		}

		// 返回 nil 表示继续遍历
		return nil
	})

	// 检查遍历过程是否出错
	if err != nil {
		log.Fatalf("遍历目录失败: %s", err)
	}

	log.Printf("共发现%d个图片", len(paths))
	log.Print("✅ 确定吗？(Y/n) [默认Y]: ")

	scanner2 := bufio.NewScanner(os.Stdin)
	scanner2.Scan()
	confirm := strings.ToLower(strings.TrimSpace(scanner2.Text()))

	// 默认回车 = Y
	if !(confirm == "" || confirm == "y" || confirm == "yes") {
		return
	}

	log.Println("开始并发压缩图片……")

	wg := errgroup.Group{}
	// 限制为核心数的一半
	if runtime.NumCPU() > 1 {
		wg.SetLimit(runtime.NumCPU() / 2)
	}
	var size0, size1 int64
	for _, path := range paths {
		wg.Go(func() error {
			defer func() {
				if err := recover(); err != nil {
					// 记录错误日志
					log.Printf("goroutine panic: %v\n", err)
					debug.PrintStack()
				}
			}()
			info, _ := os.Stat(path)
			ori := info.Size()
			atomic.AddInt64(&size0, ori)
			if err := CompressImage(path); err != nil {
				log.Printf("压缩 %s 失败: %v\n", path, err)
			} else {
				info, _ = os.Stat(path)
				log.Printf("压缩 %s 成功，压缩了%d%%\n", path, 100-int64(info.Size())*100/ori)
				atomic.AddInt64(&size1, info.Size())
			}
			return nil
		})
	}
	if err := wg.Wait(); err != nil {
		log.Fatalf("压缩图片失败: %s", err)
	}
	log.Printf("✅ 压缩完成，共压缩了 %d 个图片，压缩率 %.2f%%\n", len(paths), 100-float64(size1)/float64(size0)*100)
}
