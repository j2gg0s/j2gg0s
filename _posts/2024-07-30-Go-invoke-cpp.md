Go 支持调用 C 代码, 这种能力来自 runtime 中对调用 C 函数的封装,
[cgocall](https://github.com/golang/go/blob/go1.21.12/src/runtime/cgocall.go#L124).

调用前提的是将相关的函数暴露给 Go 代码.
来自官方的一个例子:
```go
package main

/*
#include <stdlib.h>
*/
import "C"
import (
    "fmt"
    "time"
)

func Random() int {
    return int(C.random())
}

func Seed(i int) {
    C.srandom(C.uint(i))
}

func main() {
    Seed(time.Now().Second())
    fmt.Println(Random())
}
```
在编译时增加相关参数 `go build -x -work -a` 可以看到具体的过程和现场.
不难发现, 其首先通过 `go tool cgo` 生成了相关的代码, 随后通过 clang 编译这些 C 代码.
```shell
go build -x -work -a 2>&1 | grep -A 8 "main.go$"
TERM='dumb' CGO_LDFLAGS='"-O2" "-g"' /Users/j2gg0s/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.22.4.darwin-arm64/pkg/tool/darwin_arm64/cgo -objdir $WORK/b001/ -importpath github.com/j2gg0s/gist/cgo -- -I $WORK/b001/ -O2 -g ./main.go
cd $WORK/b001
TERM='dumb' clang -I /Users/j2gg0s/go/src/github.com/j2gg0s/gist/cgo -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b001=/tmp/go-build -gno-record-gcc-switches -fno-common -I $WORK/b001/ -O2 -g -frandom-seed=nbhFK3v3q8WmHrRATnrf -o $WORK/b001/_x001.o -c _cgo_export.c
TERM='dumb' clang -I /Users/j2gg0s/go/src/github.com/j2gg0s/gist/cgo -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b001=/tmp/go-build -gno-record-gcc-switches -fno-common -I $WORK/b001/ -O2 -g -frandom-seed=nbhFK3v3q8WmHrRATnrf -o $WORK/b001/_x002.o -c main.cgo2.c
TERM='dumb' clang -I /Users/j2gg0s/go/src/github.com/j2gg0s/gist/cgo -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b001=/tmp/go-build -gno-record-gcc-switches -fno-common -I $WORK/b001/ -O2 -g -frandom-seed=nbhFK3v3q8WmHrRATnrf -o $WORK/b001/_cgo_main.o -c _cgo_main.c
cd /Users/j2gg0s/go/src/github.com/j2gg0s/gist/cgo
TERM='dumb' clang -I . -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b001=/tmp/go-build -gno-record-gcc-switches -fno-common -o $WORK/b001/_cgo_.o $WORK/b001/_cgo_main.o $WORK/b001/_x001.o $WORK/b001/_x002.o -O2 -g
TERM='dumb' /Users/j2gg0s/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.22.4.darwin-arm64/pkg/tool/darwin_arm64/cgo -dynpackage main -dynimport $WORK/b001/_cgo_.o -dynout $WORK/b001/_cgo_import.go
cat >/var/folders/3g/291pfdd54f11rlb15_hbq1dr0000gn/T/go-build3932226782/b001/importcfg << 'EOF' # internal
```

Go 并不支持调用 C++ 代码，但是我们可以通过使用 C 桥接二者.
以利用 [pokerstove](https://github.com/andrewprock/pokerstove) 来计算德州扑克胜率为例.
我们首先需要将对对象方法的调用包装成 C 兼容的函数调用.
```shell
cat bridge.cpp
#include <string>
#include <vector>

#include "pokerstove/penum/ShowdownEnumerator.h"

#include "bridge.h"

void calculateEquity(const char** _hands, int numHands, const char* _board, EquityResult* _results) {
  std::string board(_board);
  std::vector<pokerstove::CardDistribution> handsDist;

  for (int i = 0; i < numHands; i++) {
    pokerstove::CardDistribution dist;
    dist.parse(_hands[i]);
    handsDist.push_back(dist);
  }

  std::shared_ptr<pokerstove::PokerHandEvaluator> evaulator = pokerstove::PokerHandEvaluator::alloc("h");

  pokerstove::ShowdownEnumerator showdown;
  std::vector<pokerstove::EquityResult> results = showdown.calculateEquity(handsDist, pokerstove::CardSet(board), evaulator);

  for (int i = 0; i < results.size(); i++) {
    _results[i].winShares = results[i].winShares;
    _results[i].tieShares = results[i].tieShares;
  }
};
cat bridge.h
#pragma once

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
  double winShares;
  double tieShares;
} EquityResult;

void calculateEquity(const char**, int , const char*, EquityResult*);

#ifdef __cplusplus
}
#endif
```
随后在 Go 中, 我们就可以将其视作对 C 的调用.
```shell
cat bridge.go | grep "func calculateEquity" -A 25
func calculateEquity(hands []string, board string) ([]Equity, error) {
        cHands := make([]*C.char, len(hands))
        for i, hand := range hands {
                cHands[i] = C.CString(hand)
                defer C.free(unsafe.Pointer(cHands[i]))
        }

        cBoard := C.CString(board)
        defer C.free(unsafe.Pointer(cBoard))

        cResults := make([]C.EquityResult, len(hands))
        C.calculateEquity((**C.char)(unsafe.Pointer(&cHands[0])), C.int(len(hands)), cBoard, &cResults[0])

        results := make([]Equity, len(cResults))
        total := 0.0
        for i, cresult := range cResults {
                results[i].WinShares = float64(cresult.winShares)
                results[i].TieShares = float64(cresult.tieShares)
                total += results[i].WinShares + results[i].TieShares
        }
        for i, result := range results {
                results[i].Equity = (result.WinShares + result.TieShares) / total
                results[i].Total = total
        }
        return results, nil
}
```
Go 允许我们通过 `import "C"` 前的注释来控制要引入的 C 代码, 编译和链接的选项.
```shell
cat bridge.go | grep 'import "C"' -B 10
package pokerstove

// #cgo CXXFLAGS: --std=c++14 -I/Users/j2gg0s/cpp/pokerstove/src/lib
// #cgo darwin CXXFLAGS: -I/opt/homebrew/opt/boost/include
// #cgo LDFLAGS: -L/Users/j2gg0s/cpp/pokerstove/build/lib -lpeval -lpenum
// #include <bridge.h>
// #include <stdlib.h>
import "C"
```
```shell
go build -x -work -a ./cmd/pfai 2>&1 | grep clang | grep -E "bridge.cpp|penum"
TERM='dumb' clang++ -I . -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b054=/tmp/go-build -gno-record-gcc-switches -fno-common -I $WORK/b054/ -O2 -g --std=c++14 -I/Users/j2gg0s/cpp/pokerstove/src/lib -I/opt/homebrew/opt/boost/include -frandom-seed=7TpMI2L1HE_JeueLc6WL -o $WORK/b054/_x003.o -c bridge.cpp
TERM='dumb' clang++ -I ./pokerstove -fPIC -arch arm64 -pthread -fno-caret-diagnostics -Qunused-arguments -fmessage-length=0 -ffile-prefix-map=$WORK/b054=/tmp/go-build -gno-record-gcc-switches -fno-common -o $WORK/b054/_cgo_.o $WORK/b054/_cgo_main.o $WORK/b054/_x001.o $WORK/b054/_x002.o $WORK/b054/_x003.o -O2 -g -L/Users/j2gg0s/cpp/pokerstove/build/lib -lpeval -lpenum
```

对于写惯了 Go/Python/Java 的同学来说,
在 Go 中调用 C/C++ 代码的入门难点可能还是在于不习惯 C/C++ 本身的复杂的编译体系.

感谢 GPT, 我们快速简单的了解 C++ 项目的编译分为四个步骤:
- 预处理(preprocessing), 处理代码中的指令, 以 # 开头, 如 #include, #define 等
- 编译(compilation), 将预处理的代码转为汇编
- 汇编(assembly), 将汇编代码转为目标机器的代码
- 链接(linking), 将一个或多个目标文件以及所需的库链接在一起, 生成最终可执行的文件
