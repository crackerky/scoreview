import { DaySwitch } from '@/components/score/DaySwitch'
import { Timetable } from '@/components/score/Timetable'
import { requireScoreBoard } from '@/score/score.load'
import { dayHasLive, dayLabel, scoreDays } from '@/score/score.utils'

// タイムスケジュール。日ごとの時間割(縦に時刻、横にコート)を、入ってすぐ見えるように出す。

export default async function SchedulePage({ params }: { params: Promise<{ id: string }> }) {
  const { board, data } = await requireScoreBoard((await params).id)
  const days = scoreDays(board, data)
  return (
    <>
      <title>{`${board.tournament.name.ja} - タイムスケジュール`}</title>
      {days.length === 0 ? (
        <p className="sc-empty">試合の日程はまだ決まっていません。</p>
      ) : (
        <DaySwitch
          days={days.map((day) => ({
            date: day.date,
            label: dayLabel(day.date),
            live: dayHasLive(day),
          }))}
          panels={days.map((day) => (
            <Timetable key={day.date ?? 'undated'} tournamentId={board.tournament.id} day={day} />
          ))}
        />
      )}
    </>
  )
}
