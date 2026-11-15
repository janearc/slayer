-- slayer.lua: Slayer's job files in nvim, with no plugin manager.
--
-- `slayer edit` loads it for the job it opens. To have it always, put
-- `luafile /path/to/slayer.lua` in your init file, or copy it into your
-- config's lua folder and `require` it.
--
-- For a YAML file whose $schema names job-v1 it:
--   - starts the YAML language server with the job schema, when
--     yaml-language-server is on PATH, so a key's description shows on
--     hover and keys and presets complete;
--   - turns on autoread and looks at the file every second, so a save
--     from `slayer watch` or the agent appears;
--   - maps <leader>d to `:!slayer draw %`.
--
-- The schema is vim.g.slayer_schema when set (`slayer edit` sets it), or
-- what `slayer schema --where` prints.

local check_every = 1000 -- milliseconds between looks at the files on disk

-- is_job is whether a buffer is a job: a $schema naming job-v1 near its
-- top.
local function is_job(buffer)
  for _, line in ipairs(vim.api.nvim_buf_get_lines(buffer, 0, 20, false)) do
    if line:find('%$schema') and line:find('job%-v1') then
      return true
    end
  end
  return false
end

-- schema_path is the job schema's path on disk, or nil.
local function schema_path()
  if vim.g.slayer_schema and vim.g.slayer_schema ~= '' then
    return vim.g.slayer_schema
  end
  local said = vim.fn.systemlist({ 'slayer', 'schema', '--where' })
  if vim.v.shell_error == 0 and said[1] and said[1] ~= '' then
    return said[1]
  end
  return nil
end

-- attach gives a job's buffer autoread, the draw key and the server.
local function attach(buffer)
  if not is_job(buffer) then
    return
  end
  vim.bo[buffer].autoread = true
  vim.keymap.set('n', '<leader>d', ':!slayer draw %<CR>',
    { buffer = buffer, desc = 'slayer draw this job' })
  if vim.fn.executable('yaml-language-server') ~= 1 then
    return
  end
  local schema = schema_path()
  if not schema then
    return
  end
  vim.lsp.start({
    name = 'slayer-yaml',
    cmd = { 'yaml-language-server', '--stdio' },
    root_dir = vim.fs.dirname(vim.api.nvim_buf_get_name(buffer)),
    settings = {
      yaml = {
        schemas = { [schema] = { '*.yaml', '*.yml' } },
        schemaStore = { enable = false },
        hover = true,
        completion = true,
        validate = true,
      },
    },
  }, { bufnr = buffer })
end

vim.o.autoread = true
local loop = vim.uv or vim.loop
local looking = loop.new_timer()
looking:start(check_every, check_every, vim.schedule_wrap(function()
  if vim.fn.getcmdwintype() == '' then
    vim.cmd('silent! checktime')
  end
end))

vim.api.nvim_create_autocmd({ 'BufReadPost', 'BufNewFile' }, {
  pattern = { '*.yaml', '*.yml' },
  callback = function(event)
    attach(event.buf)
  end,
})

for _, buffer in ipairs(vim.api.nvim_list_bufs()) do
  if vim.api.nvim_buf_is_loaded(buffer) then
    attach(buffer)
  end
end
