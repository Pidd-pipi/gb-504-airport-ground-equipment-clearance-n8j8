import { HttpClient } from '@angular/common/http';
import { Injectable, signal } from '@angular/core';
import { forkJoin } from 'rxjs';
import { groundUnitListApi, groundUnitOccupancyApi, groundUnitSummaryApi } from '../api/ground-unit.api';
import { GroundUnit, GroundUnitSummary, UnitOccupancy } from '../types';
import { parseHttpError } from '../utils/request';

@Injectable({ providedIn: 'root' })
export class GroundUnitStore {
  readonly items = signal<GroundUnit[]>([]);
  readonly total = signal(0);
  readonly loading = signal(false);
  readonly error = signal('');
  readonly summary = signal<GroundUnitSummary>({ total: 0, states: { available: 0, inspection: 0, blocked: 0, retired: 0 }, types: {}, dispatchable: 0, unavailable: 0 });
  readonly occupancy = signal<Record<number, UnitOccupancy>>({});
  private requestVersion = 0;

  constructor(private readonly http: HttpClient) {}

  load(page = 1, pageSize = 20, state = '', search = ''): void {
    const requestVersion = ++this.requestVersion;
    this.loading.set(true);
    this.error.set('');
    forkJoin({ list: groundUnitListApi(this.http, page, pageSize, state, search), summary: groundUnitSummaryApi(this.http), occupancy: groundUnitOccupancyApi(this.http) }).subscribe({
      next: result => {
        if (requestVersion !== this.requestVersion) return;
        const occupancy: Record<number, UnitOccupancy> = {};
        for (const item of result.occupancy.items) occupancy[item.unit_id] = item;
        this.items.set(result.list.list);
        this.total.set(result.list.total);
        this.summary.set(result.summary);
        this.occupancy.set(occupancy);
        this.loading.set(false);
      },
      error: error => { if (requestVersion !== this.requestVersion) return; this.error.set(parseHttpError(error)); this.loading.set(false); },
    });
  }
}
