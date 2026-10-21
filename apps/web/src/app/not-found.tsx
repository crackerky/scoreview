import { RouteNotFoundScreen } from '@/components/ErrorScreen'

// clone-spec A 7: どのルートにも合わない URL は参照サイト(CodeIgniter)の 404 画面と同じ文面
export default function RootNotFound() {
  return <RouteNotFoundScreen />
}
