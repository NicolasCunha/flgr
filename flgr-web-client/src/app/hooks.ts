import { useDispatch, useSelector, type TypedUseSelectorHook } from 'react-redux'
import type { AppDispatch, RootState } from './store'

// Typed wrappers around the plain react-redux hooks, per Redux Toolkit's
// recommended pattern — use these instead of the untyped hooks everywhere
// else in the app.
export const useAppDispatch = useDispatch.withTypes<AppDispatch>()
export const useAppSelector: TypedUseSelectorHook<RootState> = useSelector
