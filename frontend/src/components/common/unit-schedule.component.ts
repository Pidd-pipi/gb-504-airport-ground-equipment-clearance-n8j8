import { CommonModule } from '@angular/common';
import { Component, Input } from '@angular/core';
import { MatIconModule } from '@angular/material/icon';
import { UnitOccupancy } from '../../types';

// 设备计划窗口占用徽标：周转看板与设备页共用。
// 占用 = 非完成且未撤销周转自计划时间起 90 分钟；首尾相接视为可接续。
@Component({
  selector: 'app-unit-schedule',
  standalone: true,
  imports: [CommonModule, MatIconModule],
  template: `
    <div class="unit-schedule" *ngIf="occupancy; else freeNow">
      <div class="slot busy" *ngIf="occupancy.current as current">
        <mat-icon>schedule</mat-icon>
        <span class="tag">占用中</span>
        <strong>{{ current.flight_no }}</strong>
        <small>{{ current.stand }} · {{ current.start_at | date:'MM-dd HH:mm' }}–{{ current.end_at | date:'HH:mm' }}</small>
        <small class="free-at" *ngIf="occupancy.free_at">空闲于 {{ occupancy.free_at | date:'HH:mm' }}</small>
      </div>
      <div class="slot next" *ngIf="occupancy.next as next">
        <mat-icon>event_upcoming</mat-icon>
        <span class="tag">{{ occupancy.current ? '下一段' : '即将占用' }}</span>
        <strong>{{ next.flight_no }}</strong>
        <small>{{ next.stand }} · {{ next.start_at | date:'MM-dd HH:mm' }}–{{ next.end_at | date:'HH:mm' }}</small>
      </div>
      <div class="slot free" *ngIf="!occupancy.current && !occupancy.next">
        <mat-icon>event_available</mat-icon><span class="tag">空闲</span><small>当前无计划占用，可继续排班</small>
      </div>
    </div>
    <ng-template #freeNow>
      <div class="slot free"><mat-icon>event_available</mat-icon><span class="tag">空闲</span><small>暂无计划占用</small></div>
    </ng-template>
  `,
  styles: [`
    .unit-schedule { display: flex; flex-direction: column; gap: 4px; }
    .slot { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; font-size: 12px; line-height: 1.4; }
    .slot mat-icon { font-size: 15px; width: 15px; height: 15px; }
    .slot .tag { padding: 0 6px; border-radius: 8px; font-size: 11px; }
    .slot.busy .tag { background: rgba(244, 67, 54, .14); color: #c62828; }
    .slot.busy mat-icon { color: #c62828; }
    .slot.next .tag { background: rgba(255, 152, 0, .16); color: #ef6c00; }
    .slot.next mat-icon { color: #ef6c00; }
    .slot.free .tag { background: rgba(67, 160, 71, .14); color: #2e7d32; }
    .slot.free mat-icon { color: #2e7d32; }
    .slot small { color: rgba(0, 0, 0, .62); }
    .slot .free-at { color: #c62828; font-weight: 500; }
  `],
})
export class UnitScheduleComponent {
  @Input() occupancy: UnitOccupancy | null | undefined;
}
