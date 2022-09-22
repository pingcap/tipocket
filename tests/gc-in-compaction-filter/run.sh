host=$1
port=$2

init_timeout=--thread-init-timeout=1000

for i in `seq 1 100`; do
    if [ $(($i % 2)) -eq 1 ]; then
        echo "enable compaction filter"
        mysql -h $host --port=$port -u root -e "set config tikv gc.enable-compaction-filter = true"
    else
        echo "disable compaction filter"
        mysql -h $host --port=$port -u root -e "set config tikv gc.enable-compaction-filter = false"
    fi
    sysbench $init_timeout --mysql-host=$host --mysql-port=$port --mysql-user=root --tables=128 --table-size=1000000 --threads=128 --time=600 updates run
    if [ $? -ne 0 ]; then
        echo "fail"
        return
    fi 
done
echo "success"
