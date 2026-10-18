// The probe of the gates (855b46): the smallest hooks module that can add a block within a turn. At `tool.call` it
// passes the call on and returns the tool's result as it is. It holds no model call and no command. A fixture of the
// acceptance tests, shipped by nothing and installed by none of `engram update`.

export function register(on) {
  on('tool.call', async (_$, e, next) => next(e))
}
