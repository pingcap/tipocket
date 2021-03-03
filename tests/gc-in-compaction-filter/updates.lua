require("common")

function sleep(n) os.execute("sleep " .. n) end

function get_random_zero_sum_seq(len)
    local sum = 0
    local t = {}
    for i = 1, len / 2 do
        local x = sysbench.rand.uniform(1, 255)
        table.insert(t, x)
        table.insert(t, -x)
    end
    if #t == len - 1 then table.insert(t, 0) end
    return t
end

function prepare_statements()
    prepare_begin()
    prepare_commit()
    prepare_for_each_table("delete")
    prepare_for_each_table("insert")
    prepare_for_each_table("update")
end

function too_many_processlist(con)
    local rs = con:query("show processlist")
    local busy_count = 0
    for i = 1, rs.nrows do
        local command = unpack(rs:fetch_row(), 5, 5)
        if command ~= "Sleep" then busy_count = busy_count + 1 end
    end
    rs:free()
    return busy_count >= 20
end

function get_counters(con, tid)
    local sql = "select sum(k), count(id) from sbtest" .. tid
    local rs = con:query(sql)
    local sum_k_s, count_id_s = unpack(rs:fetch_row(), 1, 2)
    rs:free()
    local sum_k = tonumber(sum_k_s)
    local count_id = tonumber(count_id_s)
    local sql = string.format("select count(k) from sbtest%u use index(k)", tid)
    local rs = con:query(sql)
    local count_k_s = unpack(rs:fetch_row(), 1, 1)
    rs:free()
    local count_k = tonumber(count_k_s)
    return sum_k, count_id, count_k
end

function thread_init()
    drv = sysbench.sql.driver()
    con = drv:connect()
    stmt = {}
    param = {}
    for t = 1, sysbench.opt.tables do
        stmt[t] = {}
        param[t] = {}
    end
    prepare_statements()

    local tid = sysbench.tid % sysbench.opt.threads + 1
    if tid <= sysbench.opt.tables then
        while too_many_processlist(con) do sleep(1) end
        sum_k, count_id, count_k = get_counters(con, tid)
        print("sbtest" .. tid .. ", sum(k): " .. sum_k .. ", count(id): " ..
                  count_id .. ", count(k): " .. count_k)
    end
    print("enable async commit for session: " .. sysbench.tid % sysbench.opt.tables)
    con:query("set session tidb_enable_async_commit = 1")
    -- print("enable follower read for session: " .. sysbench.tid % sysbench.opt.tables)
    -- con:query("set @@tidb_replica_read = 'leader-and-follower'")
end

function thread_done()
    local tid = sysbench.tid % sysbench.opt.threads + 1
    if tid <= sysbench.opt.tables then
        while too_many_processlist(con) do sleep(1) end
        local sum_k_1, count_id_1, count_k_1 = get_counters(con, tid)
        if sum_k_1 ~= sum_k or count_id_1 ~= count_id or count_k_1 ~= count_k then
            print("corrupt sbtest" .. tid .. ", sum(k): " .. sum_k ..
                      ", count(id): " .. count_id .. ", count(k): " .. count_k)
            os.exit(-1)
        end
    end
    close_statements()
    con:disconnect()
end

function event()
    begin()

    local tnum = get_table_num()
    local id_suffix = get_id()
    local rs = con:query(string.format(
                             "SELECT id,k,id_suffix FROM sbtest%u WHERE id_suffix BETWEEN %d AND %d for update",
                             tnum, id_suffix, id_suffix + sysbench.opt.modifies))
    local update_list = {}
    local del_ins_list = {}
    for i = 1, rs.nrows do
        local id, k, suffix = unpack(rs:fetch_row(), 1, rs.nfields)
        id = tonumber(id)
        k = tonumber(k)
        suffix = tonumber(suffix)

        if sysbench.rand.uniform(1, 2) % 2 == 1 then
            local t = {}
            t["id"] = id
            t["k"] = k
            t["suffix"] = suffix
            table.insert(del_ins_list, t)
        else
            table.insert(update_list, id)
        end
    end
    rs:free()

    for i = 1, #del_ins_list do
        local id = del_ins_list[i]["id"]
        param[tnum].delete[1]:set(id)
        stmt[tnum].delete:execute()
        param[tnum].insert[1]:set(id + sysbench.opt.table_size)
        param[tnum].insert[2]:set(del_ins_list[i]["k"])
        if sysbench.rand.uniform(1, 6) % 6 == 3 then
            param[tnum].insert[3]:set(get_c_value())
        else
            param[tnum].insert[3]:set("")
        end
        param[tnum].insert[4]:set(get_pad_value())
        param[tnum].insert[5]:set(del_ins_list[i]["suffix"])
        stmt[tnum].insert:execute()
    end

    local zero_sum_seq = get_random_zero_sum_seq(#update_list)
    for i = 1, #update_list do
        param[tnum].update[1]:set(zero_sum_seq[i])
        param[tnum].update[2]:set(update_list[i])
        stmt[tnum].update:execute()
    end

    commit()
end
