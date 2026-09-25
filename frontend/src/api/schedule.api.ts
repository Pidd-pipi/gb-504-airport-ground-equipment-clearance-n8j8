import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable, map } from 'rxjs';
import { ApiResponse, OccupancyBoard } from '../types';
import { API_BASE, extractData } from '../utils/request';

// 计划窗口占用看板：按 unit_ids 拉取当前/下一段 90 分钟占用；不传则返回全部设备。
export function scheduleOccupancyApi(http: HttpClient, unitIds: number[] = []): Observable<OccupancyBoard> {
  let params = new HttpParams();
  if (unitIds.length) {
    params = params.set('unit_ids', Array.from(new Set(unitIds)).join(','));
  }
  return http.get<ApiResponse<OccupancyBoard>>(`${API_BASE}/v1/ground-units/occupancy`, { params }).pipe(map(extractData));
}
