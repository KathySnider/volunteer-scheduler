'use client';

import { useParams } from 'next/navigation';
import EventDetailPage from '../../components/EventDetailPage';

export default function EventPage() {
  const params = useParams();
  return <EventDetailPage eventId={params.id} />;
}