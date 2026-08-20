import { ThemeProvider } from 'next-themes'
import { Provider } from 'react-redux'
import { RouterProvider } from 'react-router-dom'
import { store } from '@/app/store'
import { router } from '@/app/router'
import { Toaster } from '@/components/ui/sonner'

export function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem storageKey="flgr-theme">
      <Provider store={store}>
        <RouterProvider router={router} />
        <Toaster />
      </Provider>
    </ThemeProvider>
  )
}
