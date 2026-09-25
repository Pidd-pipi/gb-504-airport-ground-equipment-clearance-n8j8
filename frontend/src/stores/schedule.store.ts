import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { scheduleOccupancyApi } from '../api/schedule.api';
import { OccupancyBoard } from '../types';
import { parseHttpError } from '../utils/request';

const EMPTY_BOARD: OccupancyBoard = { reserve_minutes: 90, units: {} };

@Injectable({ providedIn: 'root' })
export class ScheduleStore {
  readonly board = signal<OccupancyBoard>(EMPTY_BOARD);
  readonly loading = signal(false);
  readonly error = signal('');
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  // 拉取全部设备的计划窗口占用；看板按 unit id 建索引供页面直查。
  load(): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    scheduleOccupancyApi(this.http, []).subscribe({
      next: board => {
        if (requestVersion !== this.requestVersion) return;
        this.board.set(board);
        this.loading.set(false);
      },
      error: error => {
        if (requestVersion !== this.requestVersion) return;
        this.error.set(parseHttpError(error));
        this.loading.set(false);
      },
    });
  }

  byUnitId(unitId: string | number) {
    return this.board().units[String(unitId)] ?? null;
  }
}
