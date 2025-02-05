#!/bin/bash

# 设置循环次数
count=20

# 循环运行 go test
for ((i=1; i<=count; i++))
do
    echo "Running test iteration $i..."
    
    # 定义日志文件路径
    log_file="/tmp/gtest_${i}.log"
    
    # 运行 go test 并将输出重定向到日志文件
    go test -run 3A > "$log_file" 2>&1
    
    # 检查测试是否通过
    if [[ $? -ne 0 ]]; then
        echo "Test failed on iteration $i"
        echo "Check the log file for details: $log_file"
        exit 1
    fi
    
    echo "Test iteration $i passed. Log saved to $log_file"
done

echo "All $count test iterations passed successfully!"