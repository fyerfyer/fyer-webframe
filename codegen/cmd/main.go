package main

import (
	"flag"
	"fmt"
	"github.com/fyerfyer/fyer-webframe/codegen/predicate_gen"
	"log"
	"os"
	"path/filepath"
)

func main() {
	input := flag.String("i", "", "input file path (e.g., ./test/user.go)")
	output := flag.String("o", "", "output directory (e.g., ./test)")
	flag.Parse()

	if *input == "" || *output == "" {
		fmt.Println("Usage: predicate-gen -i <input_file> -o <output_dir>")
		fmt.Println("Example: predicate-gen -i ./test/user.go -o ./test")
		flag.Usage()
		os.Exit(1)
	}

	// 确保输入文件存在
	if _, err := os.Stat(*input); os.IsNotExist(err) {
		log.Fatalf("input file does not exist: %s", *input)
	}

	// 规范化输出目录路径
	outputDir := filepath.Clean(*output)

	// 生成代码
	if err := predicate_gen.Generate(*input, outputDir); err != nil {
		log.Fatalf("failed to generate code: %v", err)
	}

	fmt.Printf("Code generation completed successfully!\nOutput directory: %s\n", outputDir)
}
