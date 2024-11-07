$tests = "FuzzNative","FuzzMyStruct","FuzzFuzzHeaders"
foreach ($test in $tests) {
    $file = ($test + ".txt")
    1..20 | % {
        Measure-Command -Expression { go test -json go-seminar -fuzz ("^\Q" + $test + "\E$") -run "^$" } | grep "TotalSeconds" | Out-file -filepath $file -Append
#         out-file -filepath $file -Append "\n"
        rm -Recurse -Force (".\testdata\fuzz\" + $test)
    }
}