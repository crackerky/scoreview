import { Barlow_Condensed, Dela_Gothic_One, LINE_Seed_JP, M_PLUS_2 } from 'next/font/google'

// 観戦アプリとサイトの入口(大会一覧、個人情報保護方針)の文字(Claude Design「StageCam」)。本文は
// LINE Seed JP、点数と時刻は Barlow Condensed、カードのチーム名は M PLUS 2、大きく開いた試合の点数と
// チーム名は Dela Gothic One。日本語のフォントは大きいので先読みしない(表示は代わりのフォントから
// 入れ替わる)
const bodyFont = LINE_Seed_JP({
  weight: ['400', '700', '800'],
  display: 'swap',
  preload: false,
  variable: '--live-font-body',
})
const numberFont = Barlow_Condensed({
  weight: ['600', '700'],
  subsets: ['latin'],
  display: 'swap',
  variable: '--live-font-number',
})
const cardFont = M_PLUS_2({
  weight: ['800'],
  display: 'swap',
  preload: false,
  variable: '--live-font-card',
})
const stageFont = Dela_Gothic_One({
  weight: ['400'],
  display: 'swap',
  preload: false,
  variable: '--live-font-stage',
})

/** 文字の CSS 変数を定める class(外枠の要素に付ける) */
export const liveFontClass = `live-fonts ${bodyFont.variable} ${numberFont.variable} ${cardFont.variable} ${stageFont.variable}`
