"".fnExitS STEXT size=112 args=0x0 locals=0x28 funcid=0x0                                                                                                                            [190/1827]
        0x0000 00000 (main.go:42)       TEXT    "".fnExitS(SB), ABIInternal, $48-0
        0x0000 00000 (main.go:42)       MOVD    16(g), R1
        0x0004 00004 (main.go:42)       PCDATA  $0, $-2
        0x0004 00004 (main.go:42)       MOVD    RSP, R2
        0x0008 00008 (main.go:42)       CMP     R1, R2
        0x000c 00012 (main.go:42)       BLS     100
        0x0010 00016 (main.go:42)       PCDATA  $0, $-1
        0x0010 00016 (main.go:42)       MOVD.W  R30, -48(RSP)
        0x0014 00020 (main.go:42)       MOVD    R29, -8(RSP)
        0x0018 00024 (main.go:42)       SUB     $8, RSP, R29
        0x001c 00028 (main.go:42)       MOVD    ZR, 32(RSP)
        0x0020 00032 (main.go:42)       FUNCDATA        ZR, gclocals·69c1753bd5f81501d95132d08af04464(SB)
        0x0020 00032 (main.go:42)       FUNCDATA        $1, gclocals·9fb7f0986f647f17cb53dda1484e0f7a(SB)
        0x0020 00032 (main.go:42)       FUNCDATA        $4, "".fnExitS.opendefer(SB)
        0x0020 00032 (main.go:42)       MOVB    ZR, ""..autotmp_0-9(SP)
        0x0024 00036 (main.go:43)       MOVD    $"".fnExitS.func1·f(SB), R0
        0x002c 00044 (main.go:43)       MOVD    R0, ""..autotmp_1-8(SP)
        0x0030 00048 (main.go:43)       MOVD    $1, R0
        0x0034 00052 (main.go:43)       MOVB    R0, ""..autotmp_0-9(SP)
        0x0038 00056 (main.go:44)       MOVD    ZR, 8(RSP)
        0x003c 00060 (main.go:44)       PCDATA  $1, $1
        0x003c 00060 (main.go:44)       CALL    os.Exit(SB)
        0x0040 00064 (main.go:45)       MOVB    ZR, ""..autotmp_0-9(SP)
        0x0044 00068 (main.go:45)       CALL    "".fnExitS.func1(SB)
        0x0048 00072 (main.go:45)       MOVD    -8(RSP), R29
        0x004c 00076 (main.go:45)       MOVD.P  48(RSP), R30
        0x0050 00080 (main.go:45)       RET     (R30)
        0x0054 00084 (main.go:45)       CALL    runtime.deferreturn(SB)
        0x0058 00088 (main.go:45)       MOVD    -8(RSP), R29
        0x005c 00092 (main.go:45)       MOVD.P  48(RSP), R30
        0x0060 00096 (main.go:45)       RET     (R30)
        0x0064 00100 (main.go:45)       NOP
        0x0064 00100 (main.go:42)       PCDATA  $1, $-1
        0x0064 00100 (main.go:42)       PCDATA  $0, $-2
        0x0064 00100 (main.go:42)       MOVD    R30, R3
        0x0068 00104 (main.go:42)       CALL    runtime.morestack_noctxt(SB)
        0x006c 00108 (main.go:42)       PCDATA  $0, $-1
        0x006c 00108 (main.go:42)       JMP     0
        0x0000 81 0b 40 f9 e2 03 00 91 5f 00 01 eb c9 02 00 54  ..@....._......T
        0x0010 fe 0f 1d f8 fd 83 1f f8 fd 23 00 d1 ff 13 00 f9  .........#......
        0x0020 ff 7f 00 39 00 00 00 90 00 00 00 91 e0 13 00 f9  ...9............
        0x0030 e0 03 40 b2 e0 7f 00 39 ff 07 00 f9 00 00 00 94  ..@....9........
        0x0040 ff 7f 00 39 00 00 00 94 fd 83 5f f8 fe 07 43 f8  ...9......_...C.
        0x0050 c0 03 5f d6 00 00 00 94 fd 83 5f f8 fe 07 43 f8  .._......._...C.
        0x0060 c0 03 5f d6 e3 03 1e aa 00 00 00 94 e5 ff ff 17  .._.............
        rel 36+8 t=3 "".fnExitS.func1·f+0
        rel 60+4 t=9 os.Exit+0
        rel 68+4 t=9 "".fnExitS.func1+0
        rel 84+4 t=9 runtime.deferreturn+0
        rel 104+4 t=9 runtime.morestack_noctxt+0
