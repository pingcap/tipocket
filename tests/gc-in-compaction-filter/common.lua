function init() end

if sysbench.cmdline.command == nil then
    error("Command is required. Supported commands: prepare, run, help")
end

sysbench.cmdline.options = {
    table_size = {"Number of rows per table", 10000},
    tables = {"Number of tables", 1},
    modifies = {"Row to be updated in one transaction", 5}
}

function cmd_prepare()
    local drv = sysbench.sql.driver()
    local con = drv:connect()
    local threads = sysbench.opt.threads
    for i = sysbench.tid % threads + 1, sysbench.opt.tables, threads do
        create_table(drv, con, i)
    end
end

sysbench.cmdline.commands = {
    prepare = {cmd_prepare, sysbench.cmdline.PARALLEL_COMMAND}
}

-- 20 groups, 249 characters
local c_value_template = "###########-###########-###########-" ..
                             "###########-###########-###########-" ..
                             "###########-###########-###########-" ..
                             "###########-###########-###########-" ..
                             "###########-###########-###########-" ..
                             "###########-###########-###########-" ..
                             "###########-###########"

-- 4 group, 47 characters
local pad_value_template = "###########-###########-###########-###########"

function get_c_value() return sysbench.rand.string(c_value_template) end

function get_pad_value() return sysbench.rand.string(pad_value_template) end

function create_table(drv, con, table_num)
    local id_def = "BIGINT NOT NULL"
    local id_index_def = "PRIMARY KEY"
    local engine_def = ""
    local extra_table_options = ""

    print(string.format("Creating table 'sbtest%d'...", table_num))
    local query = string.format([[
        	   CREATE TABLE IF NOT EXISTS sbtest%d(
        		   id %s,
        		   k BIGINT DEFAULT '0' NOT NULL,
        		   c CHAR(255) DEFAULT '' NOT NULL,
        		   pad CHAR(60) DEFAULT '' NOT NULL,
                   id_suffix %s,
                   PRIMARY KEY (id),
                   UNIQUE KEY (id_suffix),
                   KEY (k)
        		   ) %s %s]], table_num, id_def, id_def, engine_def,
                                extra_table_options)
    con:query(query)

    print(string.format("Inserting %d records into 'sbtest%d'",
                        sysbench.opt.table_size, table_num))

    query = "INSERT INTO sbtest" .. table_num ..
                "(id, k, c, pad, id_suffix) VALUES"

    con:bulk_insert_init(query)
    for i = 1, sysbench.opt.table_size do
        local c_val = ""
        if sysbench.rand.uniform(1, 6) % 6 == 3 then
            c_val = get_c_value()
        end
        local pad_val = get_pad_value()
        query = string.format("(%d, %d, '%s', '%s', %d)", i, sb_rand(1, 255),
                              c_val, pad_val, i)
        con:bulk_insert_next(query)
    end
    con:bulk_insert_done()
end

local t = sysbench.sql.type
local stmt_defs = {
    delete = {"DELETE FROM sbtest%u WHERE id = ?", t.INT},
    insert = {
        "INSERT INTO sbtest%u (id, k, c, pad, id_suffix) VALUES (?, ?, ?, ?, ?)",
        t.INT, t.INT, {t.CHAR, 255}, {t.CHAR, 60}, t.INT
    },
    update = {"UPDATE sbtest%u SET k = k + ? WHERE id = ?", t.INT, t.INT}
}

function prepare_begin() stmt.begin = con:prepare("BEGIN") end

function prepare_commit() stmt.commit = con:prepare("COMMIT") end

function prepare_for_each_table(key)
    for t = 1, sysbench.opt.tables do
        stmt[t][key] = con:prepare(string.format(stmt_defs[key][1], t))

        local nparam = #stmt_defs[key] - 1
        if nparam > 0 then param[t][key] = {} end
        for p = 1, nparam do
            local btype = stmt_defs[key][p + 1]
            local len
            if type(btype) == "table" then
                len = btype[2]
                btype = btype[1]
            end
            if btype == sysbench.sql.type.VARCHAR or btype ==
                sysbench.sql.type.CHAR then
                param[t][key][p] = stmt[t][key]:bind_create(btype, len)
            else
                param[t][key][p] = stmt[t][key]:bind_create(btype)
            end
        end
        if nparam > 0 then stmt[t][key]:bind_param(unpack(param[t][key])) end
    end
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
end

function close_statements()
    for t = 1, sysbench.opt.tables do
        for k, s in pairs(stmt[t]) do stmt[t][k]:close() end
    end
    if (stmt.begin ~= nil) then stmt.begin:close() end
    if (stmt.commit ~= nil) then stmt.commit:close() end
end

function thread_done()
    close_statements()
    con:disconnect()
end

function get_table_num() return sysbench.rand.uniform(1, sysbench.opt.tables) end

function get_id() return sysbench.rand.uniform(1, sysbench.opt.table_size) end

function begin() stmt.begin:execute() end

function commit() stmt.commit:execute() end

function sysbench.hooks.before_restart_event(errdesc)
    if errdesc.sql_errno == 2013 or -- CR_SERVER_LOST
    errdesc.sql_errno == 2055 or -- CR_SERVER_LOST_EXTENDED
    errdesc.sql_errno == 2006 or -- CR_SERVER_GONE_ERROR
    errdesc.sql_errno == 2011 -- CR_TCP_CONNECTION
    then
        close_statements()
        prepare_statements()
    end
end
