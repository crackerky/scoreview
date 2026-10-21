'use client'

import { type MouseEvent, useCallback, useEffect, useState } from 'react'
import type { GalleryResponse } from '@/api/public.types'
import { currentLabel, stepIndex } from './lightbox.utils'

// clone-spec A 6: div#photo.CM にサムネイルの <a><img></a> を隙間なく並べ、クリックで拡大する。
// 参照サイトは jQuery の colorbox。ここでは小さな自作のライトボックスで、前後移動(端で循環)、
// "image {current} of {total}"、閉じる(Esc とオーバーレイのクリックでも閉じる)を再現する。
// JavaScript が無効でも <a href> で原寸画像を開ける。

type Photo = GalleryResponse['gallery']['photos'][number]

function Lightbox({
  photos,
  index,
  onMove,
  onClose,
}: {
  photos: Photo[]
  index: number
  onMove: (delta: -1 | 1) => void
  onClose: () => void
}) {
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft') onMove(-1)
      if (event.key === 'ArrowRight') onMove(1)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, onMove])

  const photo = photos[index]
  if (photo === undefined) {
    return null
  }
  return (
    <>
      <button type="button" className="lightbox_overlay" aria-label="close" onClick={onClose} />
      <div className="lightbox_box" role="dialog" aria-modal="true">
        <img src={photo.fullUrl} alt="" />
        <div className="lightbox_current">{currentLabel(index, photos.length)}</div>
        <button
          type="button"
          className="lightbox_button lightbox_previous"
          aria-label="previous"
          onClick={() => onMove(-1)}
        >
          ‹
        </button>
        <button
          type="button"
          className="lightbox_button lightbox_next"
          aria-label="next"
          onClick={() => onMove(1)}
        >
          ›
        </button>
        <button
          type="button"
          className="lightbox_button lightbox_close"
          aria-label="close"
          onClick={onClose}
        >
          ×
        </button>
      </div>
    </>
  )
}

export function GalleryPhotos({ photos }: { photos: Photo[] }) {
  const [openIndex, setOpenIndex] = useState<number | null>(null)
  const close = useCallback(() => setOpenIndex(null), [])
  const move = useCallback(
    (delta: -1 | 1) =>
      setOpenIndex((current) =>
        current === null ? null : stepIndex(current, delta, photos.length),
      ),
    [photos.length],
  )
  const open = (index: number) => (event: MouseEvent<HTMLAnchorElement>) => {
    event.preventDefault()
    setOpenIndex(index)
  }

  return (
    <>
      <div className="CM" id="photo">
        {photos.map((photo, index) => (
          // biome-ignore lint/suspicious/noArrayIndexKey: 同じ URL の写真が並ぶことがあり、並びは固定
          <a key={index} href={photo.fullUrl} rel="photos" onClick={open(index)}>
            <img src={photo.thumbUrl} alt="" style={{ margin: '2px' }} />
          </a>
        ))}
      </div>
      {openIndex !== null && (
        <Lightbox photos={photos} index={openIndex} onMove={move} onClose={close} />
      )}
    </>
  )
}
