import type { GalleryResponse } from '@/api/public.types'
import { GalleryPhotos } from './GalleryPhotos'
import { assetSrc, referenceAssetMode } from './referenceAsset.utils'
import { ScvMidaM } from './ScvMidaM'

// clone-spec A 6: ギャラリー /pscore/scv_tgallery/open/{id} の #score_view の中身。
// タイトルは "< {ギャラリー名} >"(lang に関係なく山括弧の内側に空白)。ページングなし。

export function GalleryView({ data }: { data: GalleryResponse }) {
  // GalleryPhotos はクライアントコンポーネントなので、環境変数を読む差し替えはここで済ませる
  const mode = referenceAssetMode()
  const photos = data.gallery.photos.map((photo) => ({
    thumbUrl: assetSrc(photo.thumbUrl, mode),
    fullUrl: assetSrc(photo.fullUrl, mode),
  }))
  return (
    <>
      <ScvMidaM>{`< ${data.gallery.name} >`}</ScvMidaM>
      <br />
      <GalleryPhotos photos={photos} />
    </>
  )
}
