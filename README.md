# 介绍

类似 zsync 的实现，用于差分下载数据，造轮子的目的是追求

1. 简单
   简单的命令行和库接口，人性化的操作界面
2. 现代
   使用 adler 滚动哈希和 sha256 强哈希验证，使用 json 明文存储
3. 上传
   计划支持差分上传功能，可在浏览器上传部分更新
4. 并发
   并发的计算和下载

# 安装

    go get github.com/myml/msync/cmd/...

# 命令行的使用

生成.msync 文件

    msync -gen new.data > .msync.new.data

启动一个支持 range 的 http file server

    http-serve ＆

差分下载

    curl http://127.0.0.1:8080/.msync.new.data | ./msync -sync old.data -url http://127.0.0.1:8080/new.data -o down.data

## 其它参数

    ➜  msync ./msync --help
    Usage of ./msync:
    -b int
            block size kb (default 5120)
    -gen string
            generate .msync file
    -out string
            output file
    -sync string
            sync file, need url,output param
    -url string
            remote url

# 库的使用

生成块信息

    splitter := msync.NewBlockSplitter(r, blockSize)
    for {
    	block, err := splitter.Next()
    	if err != nil {
    		if errors.Is(err, io.EOF) {
    			break
    		}
    		panic(err)
    	}
    	err = encoder.Encode(b)
    	if err != nil {
    		panic(err)
    	}
    }

在本地流中查找匹配的块

    finder := msync.NewBlockFinder(f, blocks, blockSize)
    for {
    	b, err := finder.Next()
    	if err != nil {
    		if err == io.EOF {
    			break
    		}
    		panic(err)
    	}
    	existsBlock[b.Index] = b
    }

复制本地块

    downloader := msync.NewBlockReaderFromReadSeeker(f)
    for _, b := range existsBlock {
    	_, err = out.Seek(b.Offset, 0)
    	if err != nil {
    		panic(err)
    	}
    	bb := *b.Block
    	bb.Offset = b.FindOffset
    	r, err := downloader.BlockReader(&bb)
    	if err != nil {
    		panic(err)
    	}
    	h := sha256.New()
    	_, err = io.CopyN(io.MultiWriter(out, h), r, b.Length)
    	if err != nil {
    		if err == io.EOF {
    			break
    		}
    		panic(err)
    	}
    	if !bytes.Equal(h.Sum(nil), b.Sha256Sum) {
    		panic("block check")
    	}
    }

复制远程块

    downloader = msync.NewBlockReaderFromHTTP(http.DefaultClient, url)
    for i := range blocks {
    	b := blocks[i]
    	if _, exists := existsBlock[b.Index]; exists {
    		continue
    	}
    	_, err = out.Seek(b.Offset, 0)
    	if err != nil {
    		panic(err)
    	}
    	r, err := downloader.BlockReader(b)
    	if err != nil {
    		panic(err)
    	}
    	h := sha256.New()
    	_, err = io.CopyN(io.MultiWriter(out, h), r, b.Length)
    	if err != nil {
    		if err == io.EOF {
    			break
    		}
    		panic(err)
    	}
    	if !bytes.Equal(h.Sum(nil), b.Sha256Sum) {
    		panic("block check")
    	}
    }
