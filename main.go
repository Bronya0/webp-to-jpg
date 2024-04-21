package main

import (
	"bytes"
	"fmt"
	"github.com/chai2010/webp"
	"github.com/pkg/errors"
	"image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

func convertWebPToJPGWorker(path string) error {
	defer wg.Done()

	// 检查文件是否为WebP格式
	if strings.ToLower(filepath.Ext(path)) == ".webp" {
		// 读取WebP图像
		webpData, err := os.ReadFile(path)
		if err != nil {
			return errors.WithStack(err)
		}
		// 创建一个字节读取器
		reader := bytes.NewReader(webpData)

		// 解码WebP图片
		img, err := webp.Decode(reader)
		if err != nil {
			return errors.WithStack(err)
		}

		// 构造目标文件路径，将后缀改为.jpg
		filename := filepath.Base(path)
		fileNameWithoutExt := filename[:len(filename)-len(filepath.Ext(path))]
		destinationPath := filepath.Join(filepath.Dir(path), fileNameWithoutExt) + ".jpg"

		// 保存为JPEG格式
		outputFile, err := os.Create(destinationPath)
		if err != nil {
			return errors.WithStack(err)
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, img, nil)
		if err != nil {
			return errors.WithStack(err)
		}

		// 删除原文件
		if err := os.Remove(path); err != nil {
			return errors.WithStack(err)
		}

		fmt.Printf("转换: %s\n", path)
	}
	return nil

}

func convertWebPToJPG(sourceDirectory string) error {

	ch := make(chan int, 100)

	fmt.Println("开始遍历...")
	_ = filepath.Walk(sourceDirectory, func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			ch <- 0
			wg.Add(1)
			go func() {
				err := convertWebPToJPGWorker(path)
				if err != nil {
					fmt.Printf("%v处理失败: %+v\n", path, err)
				}
				<-ch
			}()
		}
		return nil
	})

	wg.Wait()
	return nil
}
func main() {
	sourceDirectory := "./" // 源文件夹路径
	st := time.Now()
	if err := convertWebPToJPG(sourceDirectory); err != nil {
		fmt.Printf("Error: %+v\n", err)
	}
	fmt.Printf("处理完成，耗时：%.4f秒\n", time.Now().Sub(st).Seconds())

	fmt.Println("按回车键退出...")
	var input string
	_, _ = fmt.Scanln(&input)
}
