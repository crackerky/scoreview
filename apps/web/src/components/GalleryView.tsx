import type { GalleryResponse } from '@/api/public.types'
import { GalleryPhotos } from './GalleryPhotos'
import { ScvMidaM } from './ScvMidaM'

// clone-spec A 6: ギャラリー /pscore/scv_tgallery/open/{id} の #score_view の中身。
// タイトルは "< {ギャラリー名} >"(lang に関係なく山括弧の内側に空白)。ページングなし。

export function GalleryView({ data }: { data: GalleryResponse }) {
  return (
    <>
      <ScvMidaM>{`< ${data.gallery.name} >`}</ScvMidaM>
      <br />
      <GalleryPhotos photos={data.gallery.photos} />
    </>
  )
}
